package pastes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/bmardale/skjul/internal/db/sqlc/sqlctest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T, withS3 bool) (*Service, *sqlctest.MockQuerier, *MockObjectStorage) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockQ := sqlctest.NewMockQuerier(ctrl)
	var mockS3 *MockObjectStorage
	var s3 ObjectStorage
	if withS3 {
		mockS3 = NewMockObjectStorage(ctrl)
		s3 = mockS3
	}
	svc := NewService(mockQ, nil, s3)
	return svc, mockQ, mockS3
}

func TestExpiresAtFromString(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"30m", 30 * time.Minute, false},
		{"1h", time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"30d", 30 * 24 * time.Hour, false},
		{"never", 0, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := expiresAtFromString(now, tt.input)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidExpiration) {
					t.Fatalf("got err %v, want ErrInvalidExpiration", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.input == "never" {
				want := now.AddDate(100, 0, 0)
				if !got.Equal(want) {
					t.Fatalf("got %v, want %v", got, want)
				}
			} else {
				if got.Sub(now) != tt.expected {
					t.Fatalf("got duration %v, want %v", got.Sub(now), tt.expected)
				}
			}
		})
	}
}

func TestCreate_Success(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()
	userID := uuid.New()
	noteID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)
	exp := now.Add(24 * time.Hour)

	mockQ.EXPECT().
		CreateNote(ctx, gomock.Any()).
		Return(sqlc.CreateNoteRow{
			ID:        noteID,
			CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			ExpiresAt: pgtype.Timestamptz{Time: exp, Valid: true},
		}, nil)

	result, err := svc.Create(ctx, userID, false,
		[]byte("title"), []byte("tn"),
		[]byte("body"), []byte("bn"),
		[]byte("key"), []byte("kn"),
		"1d", "go",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != noteID {
		t.Fatalf("got ID %v, want %v", result.ID, noteID)
	}
}

func TestCreate_DefaultLanguage(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()
	userID := uuid.New()

	mockQ.EXPECT().
		CreateNote(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, p sqlc.CreateNoteParams) (sqlc.CreateNoteRow, error) {
			if p.LanguageID.String != "plaintext" {
				t.Fatalf("got language %q, want %q", p.LanguageID.String, "plaintext")
			}
			return sqlc.CreateNoteRow{
				ID:        uuid.New(),
				CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
				ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
			}, nil
		})

	_, err := svc.Create(ctx, userID, false,
		[]byte("t"), []byte("tn"),
		[]byte("b"), []byte("bn"),
		[]byte("k"), []byte("kn"),
		"1h", "",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreate_InvalidExpiration(t *testing.T) {
	svc, _, _ := newTestService(t, false)
	ctx := context.Background()

	_, err := svc.Create(ctx, uuid.New(), false,
		[]byte("t"), []byte("tn"),
		[]byte("b"), []byte("bn"),
		[]byte("k"), []byte("kn"),
		"invalid", "",
	)
	if !errors.Is(err, ErrInvalidExpiration) {
		t.Fatalf("got err %v, want ErrInvalidExpiration", err)
	}
}

func TestCreate_DBError(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()

	mockQ.EXPECT().
		CreateNote(ctx, gomock.Any()).
		Return(sqlc.CreateNoteRow{}, errors.New("db error"))

	_, err := svc.Create(ctx, uuid.New(), false,
		[]byte("t"), []byte("tn"),
		[]byte("b"), []byte("bn"),
		[]byte("k"), []byte("kn"),
		"1h", "",
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetMetaByID_Success(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()
	noteID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)
	exp := now.Add(24 * time.Hour)

	mockQ.EXPECT().
		GetNoteMetaByID(ctx, noteID).
		Return(sqlc.GetNoteMetaByIDRow{
			ID:              noteID,
			BurnAfterRead:   true,
			CreatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
			ExpiresAt:       pgtype.Timestamptz{Time: exp, Valid: true},
			LanguageID:      pgtype.Text{String: "rust", Valid: true},
			AttachmentCount: 2,
		}, nil)

	result, err := svc.GetMetaByID(ctx, noteID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != noteID {
		t.Fatalf("got ID %v, want %v", result.ID, noteID)
	}
	if !result.BurnAfterRead {
		t.Fatal("expected BurnAfterRead to be true")
	}
	if result.LanguageID != "rust" {
		t.Fatalf("got LanguageID %q, want %q", result.LanguageID, "rust")
	}
	if result.AttachmentCount != 2 {
		t.Fatalf("got AttachmentCount %d, want 2", result.AttachmentCount)
	}
}

func TestGetMetaByID_DefaultLanguage(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()
	noteID := uuid.New()
	now := time.Now()

	mockQ.EXPECT().
		GetNoteMetaByID(ctx, noteID).
		Return(sqlc.GetNoteMetaByIDRow{
			ID:        noteID,
			CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			ExpiresAt: pgtype.Timestamptz{Time: now.Add(time.Hour), Valid: true},
			LanguageID: pgtype.Text{Valid: false},
		}, nil)

	result, err := svc.GetMetaByID(ctx, noteID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.LanguageID != "plaintext" {
		t.Fatalf("got LanguageID %q, want %q", result.LanguageID, "plaintext")
	}
}

func TestGetMetaByID_NotFound(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()

	mockQ.EXPECT().
		GetNoteMetaByID(ctx, gomock.Any()).
		Return(sqlc.GetNoteMetaByIDRow{}, pgx.ErrNoRows)

	_, err := svc.GetMetaByID(ctx, uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got err %v, want ErrNotFound", err)
	}
}

func TestGetMetaByID_DBError(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()

	mockQ.EXPECT().
		GetNoteMetaByID(ctx, gomock.Any()).
		Return(sqlc.GetNoteMetaByIDRow{}, errors.New("db error"))

	_, err := svc.GetMetaByID(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatal("should not be ErrNotFound")
	}
}

func TestListByUser_Success(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)
	exp := now.Add(24 * time.Hour)

	mockQ.EXPECT().
		ListNotesByUserID(ctx, userID).
		Return([]sqlc.ListNotesByUserIDRow{
			{
				ID:                uuid.New(),
				BurnAfterRead:     false,
				TitleCiphertext:   []byte("t"),
				TitleNonce:        []byte("tn"),
				EncryptedKey:      []byte("k"),
				EncryptedKeyNonce: []byte("kn"),
				CreatedAt:         pgtype.Timestamptz{Time: now, Valid: true},
				ExpiresAt:         pgtype.Timestamptz{Time: exp, Valid: true},
				LanguageID:        pgtype.Text{String: "go", Valid: true},
				AttachmentCount:   1,
			},
		}, nil)

	notes, err := svc.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1", len(notes))
	}
	if notes[0].LanguageID != "go" {
		t.Fatalf("got LanguageID %q, want %q", notes[0].LanguageID, "go")
	}
	if notes[0].AttachmentCount != 1 {
		t.Fatalf("got AttachmentCount %d, want 1", notes[0].AttachmentCount)
	}
}

func TestListByUser_Empty(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()
	userID := uuid.New()

	mockQ.EXPECT().
		ListNotesByUserID(ctx, userID).
		Return(nil, nil)

	notes, err := svc.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("got %d notes, want 0", len(notes))
	}
}

func TestListByUser_DBError(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()

	mockQ.EXPECT().
		ListNotesByUserID(ctx, gomock.Any()).
		Return(nil, errors.New("db error"))

	_, err := svc.ListByUser(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListByUserPaginated_FirstPage(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)
	exp := now.Add(24 * time.Hour)

	id1 := uuid.New()
	id2 := uuid.New()

	mockQ.EXPECT().
		ListNotesByUserIDPaginated(ctx, sqlc.ListNotesByUserIDPaginatedParams{
			UserID:  userID,
			Column2: uuid.Nil,
			Limit:   3,
		}).
		Return([]sqlc.ListNotesByUserIDPaginatedRow{
			{ID: id1, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}, ExpiresAt: pgtype.Timestamptz{Time: exp, Valid: true}, LanguageID: pgtype.Text{Valid: false}},
			{ID: id2, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}, ExpiresAt: pgtype.Timestamptz{Time: exp, Valid: true}, LanguageID: pgtype.Text{Valid: false}},
		}, nil)

	page, err := svc.ListByUserPaginated(ctx, userID, nil, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(page.Items))
	}
	if page.NextCursor != nil {
		t.Fatal("expected no next cursor")
	}
}

func TestListByUserPaginated_HasMore(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)
	exp := now.Add(24 * time.Hour)

	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	mockQ.EXPECT().
		ListNotesByUserIDPaginated(ctx, gomock.Any()).
		Return([]sqlc.ListNotesByUserIDPaginatedRow{
			{ID: id1, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}, ExpiresAt: pgtype.Timestamptz{Time: exp, Valid: true}, LanguageID: pgtype.Text{Valid: false}},
			{ID: id2, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}, ExpiresAt: pgtype.Timestamptz{Time: exp, Valid: true}, LanguageID: pgtype.Text{Valid: false}},
			{ID: id3, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}, ExpiresAt: pgtype.Timestamptz{Time: exp, Valid: true}, LanguageID: pgtype.Text{Valid: false}},
		}, nil)

	page, err := svc.ListByUserPaginated(ctx, userID, nil, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(page.Items))
	}
	if page.NextCursor == nil {
		t.Fatal("expected next cursor")
	}
	if *page.NextCursor != id2 {
		t.Fatalf("got cursor %v, want %v", *page.NextCursor, id2)
	}
}

func TestListByUserPaginated_WithCursor(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()
	userID := uuid.New()
	cursor := uuid.New()

	mockQ.EXPECT().
		ListNotesByUserIDPaginated(ctx, sqlc.ListNotesByUserIDPaginatedParams{
			UserID:  userID,
			Column2: cursor,
			Limit:   defaultPageLimit + 1,
		}).
		Return(nil, nil)

	page, err := svc.ListByUserPaginated(ctx, userID, &cursor, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("got %d items, want 0", len(page.Items))
	}
}

func TestListByUserPaginated_DBError(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()

	mockQ.EXPECT().
		ListNotesByUserIDPaginated(ctx, gomock.Any()).
		Return(nil, errors.New("db error"))

	_, err := svc.ListByUserPaginated(ctx, uuid.New(), nil, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteByID(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()
	userID := uuid.New()
	noteID := uuid.New()

	mockQ.EXPECT().
		DeleteNoteByIDAndUserID(ctx, sqlc.DeleteNoteByIDAndUserIDParams{
			ID:     noteID,
			UserID: userID,
		}).
		Return(nil)

	if err := svc.DeleteByID(ctx, userID, noteID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanupExpiredNotes_WithS3(t *testing.T) {
	svc, mockQ, mockS3 := newTestService(t, true)
	ctx := context.Background()

	mockQ.EXPECT().
		GetAttachmentS3KeysForExpiredNotes(ctx).
		Return([]string{"a/key1", "a/key2"}, nil)

	mockS3.EXPECT().
		DeleteObjects(ctx, []string{"a/key1", "a/key2"}).
		Return(nil)

	mockQ.EXPECT().
		DeleteExpiredNotes(ctx).
		Return(nil)

	if err := svc.CleanupExpiredNotes(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanupExpiredNotes_NoS3(t *testing.T) {
	svc, mockQ, _ := newTestService(t, false)
	ctx := context.Background()

	mockQ.EXPECT().
		DeleteExpiredNotes(ctx).
		Return(nil)

	if err := svc.CleanupExpiredNotes(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanupExpiredNotes_NoExpiredAttachments(t *testing.T) {
	svc, mockQ, _ := newTestService(t, true)
	ctx := context.Background()

	mockQ.EXPECT().
		GetAttachmentS3KeysForExpiredNotes(ctx).
		Return(nil, nil)

	mockQ.EXPECT().
		DeleteExpiredNotes(ctx).
		Return(nil)

	if err := svc.CleanupExpiredNotes(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanupExpiredNotes_S3DeleteError(t *testing.T) {
	svc, mockQ, mockS3 := newTestService(t, true)
	ctx := context.Background()

	mockQ.EXPECT().
		GetAttachmentS3KeysForExpiredNotes(ctx).
		Return([]string{"a/key1"}, nil)

	mockS3.EXPECT().
		DeleteObjects(ctx, gomock.Any()).
		Return(errors.New("s3 error"))

	err := svc.CleanupExpiredNotes(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateAttachment_Success(t *testing.T) {
	svc, mockQ, mockS3 := newTestService(t, true)
	ctx := context.Background()
	userID := uuid.New()
	noteID := uuid.New()

	mockQ.EXPECT().
		GetNoteUserID(ctx, noteID).
		Return(userID, nil)

	mockQ.EXPECT().
		CountAttachmentsByNoteID(ctx, noteID).
		Return(int64(2), nil)

	mockQ.EXPECT().
		CreateAttachment(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, p sqlc.CreateAttachmentParams) (sqlc.CreateAttachmentRow, error) {
			return sqlc.CreateAttachmentRow{
				ID:        p.ID,
				CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			}, nil
		})

	mockS3.EXPECT().
		GenerateUploadURL(ctx, gomock.Any(), int64(1024)).
		Return("https://s3.example.com/upload", nil)

	result, err := svc.CreateAttachment(ctx, userID, noteID, 1024,
		[]byte("fn"), []byte("fnn"), []byte("cn"),
		[]byte("mc"), []byte("mn"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UploadURL != "https://s3.example.com/upload" {
		t.Fatalf("got URL %q, want %q", result.UploadURL, "https://s3.example.com/upload")
	}
}

func TestCreateAttachment_NoS3(t *testing.T) {
	svc, _, _ := newTestService(t, false)
	ctx := context.Background()

	_, err := svc.CreateAttachment(ctx, uuid.New(), uuid.New(), 1024,
		[]byte("fn"), []byte("fnn"), []byte("cn"),
		[]byte("mc"), []byte("mn"),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateAttachment_NoteNotFound(t *testing.T) {
	svc, mockQ, _ := newTestService(t, true)
	ctx := context.Background()

	mockQ.EXPECT().
		GetNoteUserID(ctx, gomock.Any()).
		Return(uuid.Nil, pgx.ErrNoRows)

	_, err := svc.CreateAttachment(ctx, uuid.New(), uuid.New(), 1024,
		[]byte("fn"), []byte("fnn"), []byte("cn"),
		[]byte("mc"), []byte("mn"),
	)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got err %v, want ErrNotFound", err)
	}
}

func TestCreateAttachment_Forbidden(t *testing.T) {
	svc, mockQ, _ := newTestService(t, true)
	ctx := context.Background()
	noteID := uuid.New()
	ownerID := uuid.New()
	otherID := uuid.New()

	mockQ.EXPECT().
		GetNoteUserID(ctx, noteID).
		Return(ownerID, nil)

	_, err := svc.CreateAttachment(ctx, otherID, noteID, 1024,
		[]byte("fn"), []byte("fnn"), []byte("cn"),
		[]byte("mc"), []byte("mn"),
	)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("got err %v, want ErrForbidden", err)
	}
}

func TestCreateAttachment_LimitExceeded(t *testing.T) {
	svc, mockQ, _ := newTestService(t, true)
	ctx := context.Background()
	userID := uuid.New()
	noteID := uuid.New()

	mockQ.EXPECT().
		GetNoteUserID(ctx, noteID).
		Return(userID, nil)

	mockQ.EXPECT().
		CountAttachmentsByNoteID(ctx, noteID).
		Return(int64(maxAttachmentsPerPaste), nil)

	_, err := svc.CreateAttachment(ctx, userID, noteID, 1024,
		[]byte("fn"), []byte("fnn"), []byte("cn"),
		[]byte("mc"), []byte("mn"),
	)
	if !errors.Is(err, ErrAttachmentLimit) {
		t.Fatalf("got err %v, want ErrAttachmentLimit", err)
	}
}

func TestCreateAttachment_SizeTooLarge(t *testing.T) {
	svc, mockQ, _ := newTestService(t, true)
	ctx := context.Background()
	userID := uuid.New()
	noteID := uuid.New()

	mockQ.EXPECT().
		GetNoteUserID(ctx, noteID).
		Return(userID, nil)

	mockQ.EXPECT().
		CountAttachmentsByNoteID(ctx, noteID).
		Return(int64(0), nil)

	_, err := svc.CreateAttachment(ctx, userID, noteID, 11*1024*1024,
		[]byte("fn"), []byte("fnn"), []byte("cn"),
		[]byte("mc"), []byte("mn"),
	)
	if !errors.Is(err, ErrAttachmentSizeLimit) {
		t.Fatalf("got err %v, want ErrAttachmentSizeLimit", err)
	}
}

func TestCreateAttachment_SizeZero(t *testing.T) {
	svc, mockQ, _ := newTestService(t, true)
	ctx := context.Background()
	userID := uuid.New()
	noteID := uuid.New()

	mockQ.EXPECT().
		GetNoteUserID(ctx, noteID).
		Return(userID, nil)

	mockQ.EXPECT().
		CountAttachmentsByNoteID(ctx, noteID).
		Return(int64(0), nil)

	_, err := svc.CreateAttachment(ctx, userID, noteID, 0,
		[]byte("fn"), []byte("fnn"), []byte("cn"),
		[]byte("mc"), []byte("mn"),
	)
	if !errors.Is(err, ErrAttachmentSizeLimit) {
		t.Fatalf("got err %v, want ErrAttachmentSizeLimit", err)
	}
}

func TestCreateAttachment_UploadURLError(t *testing.T) {
	svc, mockQ, mockS3 := newTestService(t, true)
	ctx := context.Background()
	userID := uuid.New()
	noteID := uuid.New()

	mockQ.EXPECT().
		GetNoteUserID(ctx, noteID).
		Return(userID, nil)

	mockQ.EXPECT().
		CountAttachmentsByNoteID(ctx, noteID).
		Return(int64(0), nil)

	mockQ.EXPECT().
		CreateAttachment(ctx, gomock.Any()).
		Return(sqlc.CreateAttachmentRow{ID: uuid.New()}, nil)

	mockS3.EXPECT().
		GenerateUploadURL(ctx, gomock.Any(), gomock.Any()).
		Return("", errors.New("s3 error"))

	_, err := svc.CreateAttachment(ctx, userID, noteID, 1024,
		[]byte("fn"), []byte("fnn"), []byte("cn"),
		[]byte("mc"), []byte("mn"),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
