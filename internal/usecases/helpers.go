package usecase

import "github.com/itzLilix/questboard-shared/dtos"

func isValidSessionStatus(s dtos.SessionStatus) bool {
	switch s {
	case dtos.Draft, dtos.Published, dtos.Ongoing, dtos.Completed, dtos.Cancelled:
		return true
	}
	return false
}

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

var publicPresetStatuses = []dtos.SessionStatus{
	dtos.Published, dtos.Ongoing, dtos.Completed,
}

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
		}
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