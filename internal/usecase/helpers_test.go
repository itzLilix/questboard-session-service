package usecase

import (
	"testing"
	"time"

	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// inOr
// ---------------------------------------------------------------------------

func TestInOr_NilPointer_ReturnsDefault(t *testing.T) {
	assert.Equal(t, "default", inOr[string](nil, "default"))
}

func TestInOr_NonNilPointer_ReturnsDereferencedValue(t *testing.T) {
	val := "hello"
	assert.Equal(t, "hello", inOr(&val, "default"))
}

func TestInOr_ZeroValue_ReturnsZero(t *testing.T) {
	zero := 0
	assert.Equal(t, 0, inOr(&zero, 42))
}

func TestInOr_NilInt_ReturnsDefault(t *testing.T) {
	assert.Equal(t, 42, inOr[int](nil, 42))
}

func TestInOr_TimePointer(t *testing.T) {
	now := time.Now()
	fallback := time.Time{}
	assert.Equal(t, now, inOr(&now, fallback))
	assert.Equal(t, fallback, inOr[time.Time](nil, fallback))
}

// ---------------------------------------------------------------------------
// isValidSessionStatus
// ---------------------------------------------------------------------------

func TestIsValidSessionStatus(t *testing.T) {
	tests := []struct {
		name   string
		status dtos.SessionStatus
		want   bool
	}{
		{"draft", dtos.Draft, true},
		{"published", dtos.Published, true},
		{"ongoing", dtos.Ongoing, true},
		{"completed", dtos.Completed, true},
		{"cancelled", dtos.Cancelled, true},
		{"empty string", dtos.SessionStatus(""), false},
		{"unknown status", dtos.SessionStatus("archived"), false},
		{"uppercase", dtos.SessionStatus("DRAFT"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidSessionStatus(tt.status))
		})
	}
}

// ---------------------------------------------------------------------------
// hasAdvertisedChanges
// ---------------------------------------------------------------------------

func TestHasAdvertisedChanges_Empty(t *testing.T) {
	in := &SessionInput{}
	assert.False(t, hasAdvertisedChanges(in))
}

func TestHasAdvertisedChanges_OnlyTitle(t *testing.T) {
	title := "new title"
	in := &SessionInput{Title: &title}
	// Title is NOT an advertised field
	assert.False(t, hasAdvertisedChanges(in))
}

func TestHasAdvertisedChanges_OnlyDescription(t *testing.T) {
	desc := "new desc"
	in := &SessionInput{Description: &desc}
	assert.False(t, hasAdvertisedChanges(in))
}

func TestHasAdvertisedChanges_Format(t *testing.T) {
	f := dtos.Online
	in := &SessionInput{Format: &f}
	assert.True(t, hasAdvertisedChanges(in))
}

func TestHasAdvertisedChanges_SystemID(t *testing.T) {
	s := "sys-1"
	in := &SessionInput{SystemID: &s}
	assert.True(t, hasAdvertisedChanges(in))
}

func TestHasAdvertisedChanges_ScheduledAt(t *testing.T) {
	now := time.Now()
	in := &SessionInput{ScheduledAt: &now}
	assert.True(t, hasAdvertisedChanges(in))
}

func TestHasAdvertisedChanges_DurationHours(t *testing.T) {
	d := 2.5
	in := &SessionInput{DurationHours: &d}
	assert.True(t, hasAdvertisedChanges(in))
}

func TestHasAdvertisedChanges_Address(t *testing.T) {
	a := "123 Main St"
	in := &SessionInput{Address: &a}
	assert.True(t, hasAdvertisedChanges(in))
}

func TestHasAdvertisedChanges_LatLng(t *testing.T) {
	lat := 55.75
	in := &SessionInput{Lat: &lat}
	assert.True(t, hasAdvertisedChanges(in))

	lng := 37.61
	in2 := &SessionInput{Lng: &lng}
	assert.True(t, hasAdvertisedChanges(in2))
}

func TestHasAdvertisedChanges_MaxSeats(t *testing.T) {
	seats := int16(5)
	in := &SessionInput{MaxSeats: &seats}
	assert.True(t, hasAdvertisedChanges(in))
}

func TestHasAdvertisedChanges_Price(t *testing.T) {
	price := 100.0
	in := &SessionInput{Price: &price}
	assert.True(t, hasAdvertisedChanges(in))
}

func TestHasAdvertisedChanges_Availability(t *testing.T) {
	a := dtos.Open
	in := &SessionInput{Availability: &a}
	assert.True(t, hasAdvertisedChanges(in))
}

func TestHasAdvertisedChanges_Multiple(t *testing.T) {
	f := dtos.Offline
	price := 50.0
	in := &SessionInput{Format: &f, Price: &price}
	assert.True(t, hasAdvertisedChanges(in))
}

// ---------------------------------------------------------------------------
// resolveStatusFilter
// ---------------------------------------------------------------------------

func TestResolveStatusFilter_EmptyRaw_DefaultsToPublicPreset(t *testing.T) {
	result := resolveStatusFilter(nil, dtos.ScopeCatalog, false)
	assert.ElementsMatch(t, []dtos.SessionStatus{dtos.Published, dtos.Ongoing, dtos.Completed}, result)
}

func TestResolveStatusFilter_PublicPreset_Explicit(t *testing.T) {
	result := resolveStatusFilter([]string{dtos.StatusPresetPublic}, dtos.ScopeCatalog, false)
	assert.ElementsMatch(t, []dtos.SessionStatus{dtos.Published, dtos.Ongoing, dtos.Completed}, result)
}

func TestResolveStatusFilter_CatalogScope_IgnoresDraftCancelled(t *testing.T) {
	raw := []string{string(dtos.Draft), string(dtos.Cancelled), string(dtos.Published)}
	result := resolveStatusFilter(raw, dtos.ScopeCatalog, false)
	assert.ElementsMatch(t, []dtos.SessionStatus{dtos.Published}, result)
}

func TestResolveStatusFilter_MasteringScope_ViewerAllowsDraft(t *testing.T) {
	raw := []string{string(dtos.Draft), string(dtos.Published)}
	result := resolveStatusFilter(raw, dtos.ScopeMastering, true)
	assert.ElementsMatch(t, []dtos.SessionStatus{dtos.Draft, dtos.Published}, result)
}

func TestResolveStatusFilter_MasteringScope_ViewerAllowsCancelled(t *testing.T) {
	raw := []string{string(dtos.Cancelled)}
	result := resolveStatusFilter(raw, dtos.ScopeMastering, true)
	assert.ElementsMatch(t, []dtos.SessionStatus{dtos.Cancelled}, result)
}

func TestResolveStatusFilter_MasteringScope_NotViewer_IgnoresDraft(t *testing.T) {
	raw := []string{string(dtos.Draft), string(dtos.Published)}
	result := resolveStatusFilter(raw, dtos.ScopeMastering, false)
	assert.ElementsMatch(t, []dtos.SessionStatus{dtos.Published}, result)
}

func TestResolveStatusFilter_PlayingScope_ViewerAllowsAll(t *testing.T) {
	raw := []string{string(dtos.Draft), string(dtos.Cancelled), string(dtos.Published), string(dtos.Ongoing), string(dtos.Completed)}
	result := resolveStatusFilter(raw, dtos.ScopePlaying, true)
	assert.ElementsMatch(t, []dtos.SessionStatus{dtos.Draft, dtos.Cancelled, dtos.Published, dtos.Ongoing, dtos.Completed}, result)
}

func TestResolveStatusFilter_PlayingScope_NotViewer_FiltersProtected(t *testing.T) {
	raw := []string{string(dtos.Draft), string(dtos.Cancelled), string(dtos.Published)}
	result := resolveStatusFilter(raw, dtos.ScopePlaying, false)
	assert.ElementsMatch(t, []dtos.SessionStatus{dtos.Published}, result)
}

func TestResolveStatusFilter_Deduplicates(t *testing.T) {
	raw := []string{string(dtos.Published), string(dtos.Published), dtos.StatusPresetPublic}
	result := resolveStatusFilter(raw, dtos.ScopeCatalog, false)
	// published appears in raw twice + inside public preset, but should be deduped
	assert.ElementsMatch(t, []dtos.SessionStatus{dtos.Published, dtos.Ongoing, dtos.Completed}, result)
}

func TestResolveStatusFilter_UnknownStatus_Ignored(t *testing.T) {
	raw := []string{"nonexistent", string(dtos.Published)}
	result := resolveStatusFilter(raw, dtos.ScopeCatalog, false)
	assert.ElementsMatch(t, []dtos.SessionStatus{dtos.Published}, result)
}
