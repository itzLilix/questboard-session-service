package usecase

import (
	"fmt"

	"github.com/itzLilix/questboard-shared/dtos"
)

func validateCreateSession(in *CreateSessionInput) error {
	if in.Title == "" || in.Format == "" || in.SystemID == "" || in.MaxSeats <= 0 {
		return fmt.Errorf("%w: missing required field", ErrInvalidData)
	}
	if in.MaxSeats > 50 {
		return fmt.Errorf("%w: maxSeats must be <= 50", ErrInvalidData)
	}
	if in.Price < 0 {
		return fmt.Errorf("%w: price must be >= 0", ErrInvalidData)
	}
	if in.DurationHours != nil && *in.DurationHours <= 0 {
		return fmt.Errorf("%w: durationHours must be > 0", ErrInvalidData)
	}
	if in.Format != dtos.Online && in.Format != dtos.Offline {
		return fmt.Errorf("%w: invalid format", ErrInvalidData)
	}
	if in.Availability != "" &&
		in.Availability != dtos.Open &&
		in.Availability != dtos.Application &&
		in.Availability != dtos.Private {
		return fmt.Errorf("%w: invalid availability", ErrInvalidData)
	}
	return nil
}

func isValidSessionStatus(s dtos.SessionStatus) bool {
	switch s {
	case dtos.Draft, dtos.Published, dtos.Ongoing, dtos.Completed, dtos.Cancelled:
		return true
	}
	return false
}

// hasAdvertisedChanges returns true if the edit touches any field that is part
// of what players see before joining. Used both for terminal-state lockout and
// to decide whether a notification should fire.
func hasAdvertisedChanges(in *EditSessionInput) bool {
	return in.Format != nil ||
		in.SystemID != nil ||
		in.ScheduledAt != nil ||
		in.DurationHours != nil ||
		in.Address != nil ||
		in.Lat != nil ||
		in.Lng != nil ||
		in.MaxSeats != nil ||
		in.Price != nil ||
		in.Availability != nil
}

// publicPresetStatuses is what dtos.StatusPresetPublic expands to.
var publicPresetStatuses = []dtos.SessionStatus{
	dtos.Published, dtos.Ongoing, dtos.Completed,
}

// resolveStatusFilter expands the "public" preset, dedupes, and applies the
// scope/target allowlist (silently dropping disallowed values to avoid leaky
// existence checks).
//
// If raw is empty, defaults to ["public"] (drafts/cancelled never appear on
// profile views by default).
//
// Allowlist:
//   - catalog or target!=viewer: {published, ongoing, completed} only
//   - mastering/playing with target=viewer: any status allowed
func resolveStatusFilter(raw []string, scope dtos.SessionScope, targetIsViewer bool) []dtos.SessionStatus {
	if len(raw) == 0 {
		raw = []string{dtos.StatusPresetPublic}
	}

	allowAll := targetIsViewer && (scope == dtos.ScopeMastering || scope == dtos.ScopePlaying)

	seen := make(map[dtos.SessionStatus]struct{}, 5)
	add := func(s dtos.SessionStatus) {
		switch s {
		case dtos.Published, dtos.Ongoing, dtos.Completed:
			seen[s] = struct{}{}
		case dtos.Draft, dtos.Cancelled:
			if allowAll {
				seen[s] = struct{}{}
			}
			// else silently drop
		}
		// unknown values silently dropped
	}

	for _, v := range raw {
		if v == dtos.StatusPresetPublic {
			for _, s := range publicPresetStatuses {
				add(s)
			}
			continue
		}
		add(dtos.SessionStatus(v))
	}

	out := make([]dtos.SessionStatus, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	return out
}
