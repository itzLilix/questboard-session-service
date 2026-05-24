package usecase

import (
	"testing"
	"time"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helper to make pointers inline
// ---------------------------------------------------------------------------

func ptr[T any](v T) *T { return &v }

// ---------------------------------------------------------------------------
// validateSession
// ---------------------------------------------------------------------------

func TestValidateSession_EmptyInput_OK(t *testing.T) {
	assert.NoError(t, validateSession(&SessionInput{}))
}

func TestValidateSession_AllFieldsValid(t *testing.T) {
	now := time.Now().Add(time.Hour)
	in := &SessionInput{
		Title:         ptr("A valid title"),
		Description:   ptr("A valid description"),
		MaxSeats:      ptr(int16(6)),
		Price:         ptr(10.0),
		DurationHours: ptr(2.5),
		ScheduledAt:   &now,
		Format:        ptr(dtos.Online),
		Availability:  ptr(dtos.Open),
	}
	assert.NoError(t, validateSession(in))
}

// --- Title ---

func TestValidateSession_TitleExactly100_OK(t *testing.T) {
	title := make([]byte, 100)
	for i := range title {
		title[i] = 'a'
	}
	in := &SessionInput{Title: ptr(string(title))}
	assert.NoError(t, validateSession(in))
}

func TestValidateSession_TitleTooLong(t *testing.T) {
	title := make([]byte, 101)
	for i := range title {
		title[i] = 'a'
	}
	in := &SessionInput{Title: ptr(string(title))}
	err := validateSession(in)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "title")
}

// --- Description ---

func TestValidateSession_DescriptionExactly2000_OK(t *testing.T) {
	desc := make([]byte, 2000)
	for i := range desc {
		desc[i] = 'x'
	}
	in := &SessionInput{Description: ptr(string(desc))}
	assert.NoError(t, validateSession(in))
}

func TestValidateSession_DescriptionTooLong(t *testing.T) {
	desc := make([]byte, 2001)
	for i := range desc {
		desc[i] = 'x'
	}
	in := &SessionInput{Description: ptr(string(desc))}
	err := validateSession(in)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "description")
}

// --- MaxSeats ---

func TestValidateSession_MaxSeatsZero(t *testing.T) {
	in := &SessionInput{MaxSeats: ptr(int16(0))}
	err := validateSession(in)
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestValidateSession_MaxSeatsNegative(t *testing.T) {
	in := &SessionInput{MaxSeats: ptr(int16(-1))}
	err := validateSession(in)
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestValidateSession_MaxSeats1_OK(t *testing.T) {
	in := &SessionInput{MaxSeats: ptr(int16(1))}
	assert.NoError(t, validateSession(in))
}

func TestValidateSession_MaxSeats50_OK(t *testing.T) {
	in := &SessionInput{MaxSeats: ptr(int16(50))}
	assert.NoError(t, validateSession(in))
}

func TestValidateSession_MaxSeats51_Error(t *testing.T) {
	in := &SessionInput{MaxSeats: ptr(int16(51))}
	err := validateSession(in)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "maxSeats")
}

// --- Price ---

func TestValidateSession_PriceZero_OK(t *testing.T) {
	in := &SessionInput{Price: ptr(0.0)}
	assert.NoError(t, validateSession(in))
}

func TestValidateSession_PriceNegative(t *testing.T) {
	in := &SessionInput{Price: ptr(-1.0)}
	err := validateSession(in)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "price")
}

func TestValidateSession_PricePositive_OK(t *testing.T) {
	in := &SessionInput{Price: ptr(500.0)}
	assert.NoError(t, validateSession(in))
}

// --- DurationHours ---

func TestValidateSession_DurationZero(t *testing.T) {
	in := &SessionInput{DurationHours: ptr(0.0)}
	err := validateSession(in)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "durationHours")
}

func TestValidateSession_DurationNegative(t *testing.T) {
	in := &SessionInput{DurationHours: ptr(-1.0)}
	err := validateSession(in)
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestValidateSession_DurationPositive_OK(t *testing.T) {
	in := &SessionInput{DurationHours: ptr(3.5)}
	assert.NoError(t, validateSession(in))
}

// --- ScheduledAt ---

func TestValidateSession_ScheduledAtZeroTime(t *testing.T) {
	zero := time.Time{}
	in := &SessionInput{ScheduledAt: &zero}
	err := validateSession(in)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "scheduledAt")
}

func TestValidateSession_ScheduledAtValid_OK(t *testing.T) {
	t1 := time.Now().Add(24 * time.Hour)
	in := &SessionInput{ScheduledAt: &t1}
	assert.NoError(t, validateSession(in))
}

// --- Format ---

func TestValidateSession_FormatOnline_OK(t *testing.T) {
	in := &SessionInput{Format: ptr(dtos.Online)}
	assert.NoError(t, validateSession(in))
}

func TestValidateSession_FormatOffline_OK(t *testing.T) {
	in := &SessionInput{Format: ptr(dtos.Offline)}
	assert.NoError(t, validateSession(in))
}

func TestValidateSession_FormatInvalid(t *testing.T) {
	in := &SessionInput{Format: ptr(dtos.SessionFormat("hybrid"))}
	err := validateSession(in)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "format")
}

// --- Availability ---

func TestValidateSession_AvailabilityOpen_OK(t *testing.T) {
	in := &SessionInput{Availability: ptr(dtos.Open)}
	assert.NoError(t, validateSession(in))
}

func TestValidateSession_AvailabilityApplication_OK(t *testing.T) {
	in := &SessionInput{Availability: ptr(dtos.Application)}
	assert.NoError(t, validateSession(in))
}

func TestValidateSession_AvailabilityPrivate_OK(t *testing.T) {
	in := &SessionInput{Availability: ptr(dtos.Private)}
	assert.NoError(t, validateSession(in))
}

func TestValidateSession_AvailabilityInvalid(t *testing.T) {
	in := &SessionInput{Availability: ptr(dtos.SessionAvailability("invite"))}
	err := validateSession(in)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "availability")
}

// ---------------------------------------------------------------------------
// isValidSessionSort
// ---------------------------------------------------------------------------

func TestIsValidSessionSort(t *testing.T) {
	tests := []struct {
		sort dtos.SessionListSort
		want bool
	}{
		{dtos.SortSessionScheduledAt, true},
		{dtos.SortSessionCreatedAt, true},
		{dtos.SortSessionPrice, true},
		{dtos.SortSessionTitle, true},
		{dtos.SortSessionSystem, true},
		{dtos.SessionListSort(""), false},
		{dtos.SessionListSort("popularity"), false},
		{dtos.SessionListSort("SCHEDULED_AT"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.sort), func(t *testing.T) {
			assert.Equal(t, tt.want, isValidSessionSort(tt.sort))
		})
	}
}

// ---------------------------------------------------------------------------
// validateListSessions
// ---------------------------------------------------------------------------

func authenticatedViewer(id string) *entities.Viewer {
	return &entities.Viewer{UserID: id}
}

func TestValidateListSessions_MinimalCatalog(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)

	assert.Equal(t, dtos.ScopeCatalog, p.Scope)
	assert.Equal(t, dtos.SortSessionScheduledAt, p.Sort)
	assert.Equal(t, dtos.SortAsc, p.SortOrder)
	assert.Equal(t, 20, p.Limit)
	assert.ElementsMatch(t, []dtos.SessionStatus{dtos.Published, dtos.Ongoing, dtos.Completed}, p.Status)
}

// --- scope ---

func TestValidateListSessions_UnknownScope(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Scope: "foo"}

	_, err := validateListSessions(in, v)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "scope")
}

// --- mastering scope ---

func TestValidateListSessions_MasteringScope_NoMasterID_UsesViewer(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Scope: dtos.ScopeMastering}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)

	assert.Equal(t, "user-1", p.MasterID)
	assert.True(t, p.TargetIsViewer)
	assert.Equal(t, dtos.SortSessionCreatedAt, p.Sort) // non-catalog default
}

func TestValidateListSessions_MasteringScope_ExplicitMasterID_SameAsViewer(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Scope: dtos.ScopeMastering, MasterID: "user-1"}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)

	assert.Equal(t, "user-1", p.MasterID)
	assert.True(t, p.TargetIsViewer)
}

func TestValidateListSessions_MasteringScope_DifferentMasterID(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Scope: dtos.ScopeMastering, MasterID: "user-2"}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)

	assert.Equal(t, "user-2", p.MasterID)
	assert.False(t, p.TargetIsViewer)
}

func TestValidateListSessions_MasteringScope_Unauthenticated_NoMasterID(t *testing.T) {
	var v *entities.Viewer // nil = unauthenticated
	in := &ListSessionsInput{Scope: dtos.ScopeMastering}

	_, err := validateListSessions(in, v)
	assert.ErrorIs(t, err, ErrForbidden)
}

// --- playing scope ---

func TestValidateListSessions_PlayingScope_NoPlayerID_UsesViewer(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Scope: dtos.ScopePlaying}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)

	assert.Equal(t, "user-1", p.PlayerID)
	assert.True(t, p.TargetIsViewer)
}

func TestValidateListSessions_PlayingScope_ExplicitPlayerID(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Scope: dtos.ScopePlaying, PlayerID: "user-2"}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)

	assert.Equal(t, "user-2", p.PlayerID)
	assert.False(t, p.TargetIsViewer)
}

func TestValidateListSessions_PlayingScope_Unauthenticated_NoPlayerID(t *testing.T) {
	var v *entities.Viewer
	in := &ListSessionsInput{Scope: dtos.ScopePlaying}

	_, err := validateListSessions(in, v)
	assert.ErrorIs(t, err, ErrForbidden)
}

// --- format / type ---

func TestValidateListSessions_ValidFormat(t *testing.T) {
	v := authenticatedViewer("user-1")

	for _, f := range []string{"online", "offline"} {
		in := &ListSessionsInput{Format: f}
		p, err := validateListSessions(in, v)
		require.NoError(t, err)
		assert.Equal(t, dtos.SessionFormat(f), p.Format)
	}
}

func TestValidateListSessions_InvalidFormat(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Format: "hybrid"}

	_, err := validateListSessions(in, v)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "format")
}

func TestValidateListSessions_ValidType(t *testing.T) {
	v := authenticatedViewer("user-1")

	for _, tp := range []string{"oneshot", "campaign"} {
		in := &ListSessionsInput{Type: tp}
		p, err := validateListSessions(in, v)
		require.NoError(t, err)
		assert.Equal(t, dtos.SessionType(tp), p.Type)
	}
}

func TestValidateListSessions_InvalidType(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Type: "tournament"}

	_, err := validateListSessions(in, v)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "type")
}

// --- game systems include/exclude & dedup ---

func TestValidateListSessions_SystemsIncluded(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{GSIncluded: []string{"sys-1", "sys-2"}}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"sys-1", "sys-2"}, p.SystemsIn)
	assert.Empty(t, p.SystemsEx)
}

func TestValidateListSessions_SystemsExcluded(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{GSExcluded: []string{"sys-3"}}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Empty(t, p.SystemsIn)
	assert.ElementsMatch(t, []string{"sys-3"}, p.SystemsEx)
}

func TestValidateListSessions_SystemsDedup_IncludeWins(t *testing.T) {
	v := authenticatedViewer("user-1")
	// same ID in both included and excluded — included is processed first so it wins
	in := &ListSessionsInput{
		GSIncluded: []string{"sys-1"},
		GSExcluded: []string{"sys-1"},
	}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"sys-1"}, p.SystemsIn)
	assert.Empty(t, p.SystemsEx)
}

// --- sort ---

func TestValidateListSessions_ExplicitSort(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Sort: "price"}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, dtos.SortSessionPrice, p.Sort)
}

func TestValidateListSessions_InvalidSort(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Sort: "popularity"}

	_, err := validateListSessions(in, v)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "sort")
}

// --- sort order ---

func TestValidateListSessions_SortOrderASC(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{SortOrder: "ASC"}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, dtos.SortAsc, p.SortOrder)
}

func TestValidateListSessions_SortOrderDESC(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{SortOrder: "DESC"}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, dtos.SortDesc, p.SortOrder)
}

func TestValidateListSessions_SortOrderLowercase(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{SortOrder: "asc"}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, dtos.SortAsc, p.SortOrder)
}

func TestValidateListSessions_SortOrderInvalid(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{SortOrder: "RANDOM"}

	_, err := validateListSessions(in, v)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "order")
}

// --- price range ---

func TestValidateListSessions_PriceMinNegative(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{PriceMin: ptr(-1.0)}

	_, err := validateListSessions(in, v)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "priceMin")
}

func TestValidateListSessions_PriceMaxNegative(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{PriceMax: ptr(-5.0)}

	_, err := validateListSessions(in, v)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "priceMax")
}

func TestValidateListSessions_PriceMinExceedsMax(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{PriceMin: ptr(100.0), PriceMax: ptr(50.0)}

	_, err := validateListSessions(in, v)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "priceMin must be <= priceMax")
}

func TestValidateListSessions_PriceRangeValid(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{PriceMin: ptr(10.0), PriceMax: ptr(100.0)}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, 10.0, *p.PriceMin)
	assert.Equal(t, 100.0, *p.PriceMax)
}

func TestValidateListSessions_PriceEqualMinMax_OK(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{PriceMin: ptr(50.0), PriceMax: ptr(50.0)}

	_, err := validateListSessions(in, v)
	assert.NoError(t, err)
}

// --- date range ---

func TestValidateListSessions_DateFromAfterDateTo(t *testing.T) {
	v := authenticatedViewer("user-1")
	now := time.Now()
	in := &ListSessionsInput{
		DateFrom: ptr(now.Add(time.Hour)),
		DateTo:   ptr(now),
	}

	_, err := validateListSessions(in, v)
	assert.ErrorIs(t, err, ErrInvalidData)
	assert.Contains(t, err.Error(), "dateFrom must be < dateTo")
}

func TestValidateListSessions_DateRangeValid(t *testing.T) {
	v := authenticatedViewer("user-1")
	now := time.Now()
	in := &ListSessionsInput{
		DateFrom: &now,
		DateTo:   ptr(now.Add(24 * time.Hour)),
	}

	_, err := validateListSessions(in, v)
	assert.NoError(t, err)
}

func TestValidateListSessions_DateSameFromTo(t *testing.T) {
	v := authenticatedViewer("user-1")
	now := time.Now()
	in := &ListSessionsInput{
		DateFrom: &now,
		DateTo:   &now,
	}

	_, err := validateListSessions(in, v)
	assert.ErrorIs(t, err, ErrInvalidData)
}

// --- limit ---

func TestValidateListSessions_LimitDefault(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, 20, p.Limit)
}

func TestValidateListSessions_LimitZero_DefaultsTo20(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Limit: 0}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, 20, p.Limit)
}

func TestValidateListSessions_LimitNegative_DefaultsTo20(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Limit: -5}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, 20, p.Limit)
}

func TestValidateListSessions_LimitOver100_ClampedTo100(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Limit: 200}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, 100, p.Limit)
}

func TestValidateListSessions_Limit50_Accepted(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Limit: 50}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, 50, p.Limit)
}

// --- passthrough fields ---

func TestValidateListSessions_PassthroughFields(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{
		Search:   "dnd",
		City:     "Moscow",
		FreeSeats: 2,
		Cursor:   "abc123",
	}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, "dnd", p.Search)
	assert.Equal(t, "Moscow", p.City)
	assert.Equal(t, 2, p.FreeSeats)
	assert.Equal(t, "abc123", p.Cursor)
}

// --- mastering scope with auth sets default sort to created_at ---

func TestValidateListSessions_DefaultSort_Catalog_ScheduledAt(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Scope: dtos.ScopeCatalog}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, dtos.SortSessionScheduledAt, p.Sort)
}

func TestValidateListSessions_DefaultSort_Mastering_CreatedAt(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Scope: dtos.ScopeMastering}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, dtos.SortSessionCreatedAt, p.Sort)
}

func TestValidateListSessions_DefaultSort_Playing_CreatedAt(t *testing.T) {
	v := authenticatedViewer("user-1")
	in := &ListSessionsInput{Scope: dtos.ScopePlaying}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, dtos.SortSessionCreatedAt, p.Sort)
}

// --- mastering scope, unauthenticated, but explicit master ID ---

func TestValidateListSessions_MasteringScope_Unauthenticated_WithExplicitID_OK(t *testing.T) {
	var v *entities.Viewer // nil
	in := &ListSessionsInput{Scope: dtos.ScopeMastering, MasterID: "user-2"}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, "user-2", p.MasterID)
	assert.False(t, p.TargetIsViewer)
}

func TestValidateListSessions_PlayingScope_Unauthenticated_WithExplicitID_OK(t *testing.T) {
	var v *entities.Viewer
	in := &ListSessionsInput{Scope: dtos.ScopePlaying, PlayerID: "user-2"}

	p, err := validateListSessions(in, v)
	require.NoError(t, err)
	assert.Equal(t, "user-2", p.PlayerID)
	assert.False(t, p.TargetIsViewer)
}
