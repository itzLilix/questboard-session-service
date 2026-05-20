package usecase

import (
	"fmt"
	"strings"

	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/dtos"
)

func validateSession(in *SessionInput) error {
	if in.MaxSeats != nil && *in.MaxSeats <= 0 {
		return fmt.Errorf("%w: missing required field", ErrInvalidData)
	}
	if in.Title != nil && len(*in.Title) > 100 {
		return fmt.Errorf("%w: title must be <= 100 characters", ErrInvalidData)
	}
	if in.Description != nil && len(*in.Description) > 2000 {
		return fmt.Errorf("%w: description must be <= 2000 characters", ErrInvalidData)
	}
	if in.MaxSeats != nil{
		if *in.MaxSeats <= 0 {
			return fmt.Errorf("%w: maxSeats must be > 0", ErrInvalidData)
		}
		if *in.MaxSeats > 50 {
			return fmt.Errorf("%w: maxSeats must be <= 50", ErrInvalidData)
		}
	}
	if in.Price != nil && *in.Price < 0 {
		return fmt.Errorf("%w: price must be >= 0", ErrInvalidData)
	}
	if in.DurationHours != nil && *in.DurationHours <= 0 {
		return fmt.Errorf("%w: durationHours must be > 0", ErrInvalidData)
	}
	if in.ScheduledAt != nil && in.ScheduledAt.IsZero() {
		return fmt.Errorf("%w: scheduledAt must be a valid date", ErrInvalidData)
	}
	if in.Format != nil {
		if *in.Format != dtos.Online && *in.Format != dtos.Offline {
			return fmt.Errorf("%w: invalid format", ErrInvalidData)
		}
	}
	if in.Availability != nil {
		if *in.Availability != dtos.Open && *in.Availability != dtos.Application && *in.Availability != dtos.Private {
			return fmt.Errorf("%w: invalid availability", ErrInvalidData)
		}
	}
	return nil
}

func validateListSessions(in *ListSessionsInput) (infrastructure.ListSessionsParams, error) {
	// --- scope ---------------------------------------------------------------
	scope := in.Scope
	if scope == "" {
		scope = dtos.ScopeCatalog
	}
	if scope != dtos.ScopeCatalog && scope != dtos.ScopeMastering && scope != dtos.ScopePlaying {
		return infrastructure.ListSessionsParams{}, fmt.Errorf("%w: unknown scope %q", ErrInvalidData, scope)
	}

	// --- target user + targetIsViewer ----------------------------------------
	var masterID, playerID string
	targetIsViewer := false
	switch scope {
	case dtos.ScopeMastering:
		if in.MasterID != "" {
			masterID = in.MasterID
			targetIsViewer = in.Viewer.Is(masterID)
		} else {
			if !in.Viewer.IsAuthenticated() {
				return infrastructure.ListSessionsParams{}, ErrForbidden
			}
			masterID = in.Viewer.UserID
			targetIsViewer = true
		}
	case dtos.ScopePlaying:
		if in.PlayerID != "" {
			playerID = in.PlayerID
			targetIsViewer = in.Viewer.Is(playerID)
		} else {
			if !in.Viewer.IsAuthenticated() {
				return infrastructure.ListSessionsParams{}, ErrForbidden
			}
			playerID = in.Viewer.UserID
			targetIsViewer = true
		}
	}

	// --- format / type strings → typed enums ---------------------------------
	format := dtos.SessionFormat(in.Format)
	if in.Format != "" && format != dtos.Online && format != dtos.Offline {
		return infrastructure.ListSessionsParams{}, fmt.Errorf("%w: invalid format %q", ErrInvalidData, in.Format)
	}
	stype := dtos.SessionType(in.Type)
	if in.Type != "" && stype != dtos.OneshotType && stype != dtos.CampaignType {
		return infrastructure.ListSessionsParams{}, fmt.Errorf("%w: invalid type %q", ErrInvalidData, in.Type)
	}

	// --- sort key ------------------------------------------------------------
	var sort dtos.SessionListSort
	if in.Sort == "" {
		if scope == dtos.ScopeCatalog {
			sort = dtos.SortSessionScheduledAt
		} else {
			sort = dtos.SortSessionCreatedAt
		}
	} else {
		sort = dtos.SessionListSort(in.Sort)
		if !isValidSessionSort(sort) {
			return infrastructure.ListSessionsParams{}, fmt.Errorf("%w: invalid sort %q", ErrInvalidData, in.Sort)
		}
	}

	// --- sort order ----------------------------------------------------------
	var order dtos.SortOrder
	switch strings.ToUpper(in.SortOrder) {
	case "":
		if scope == dtos.ScopeCatalog {
			order = dtos.SortAsc
		} else {
			order = dtos.SortDesc
		}
	case "ASC":
		order = dtos.SortAsc
	case "DESC":
		order = dtos.SortDesc
	default:
		return infrastructure.ListSessionsParams{}, fmt.Errorf("%w: invalid order %q", ErrInvalidData, in.SortOrder)
	}

	// --- price and date range check ------------------------------------------
	if in.PriceMin != nil && *in.PriceMin < 0 {
		return infrastructure.ListSessionsParams{}, fmt.Errorf("%w: priceMin must be >= 0", ErrInvalidData)
	}
	if in.PriceMax != nil && *in.PriceMax < 0 {
		return infrastructure.ListSessionsParams{}, fmt.Errorf("%w: priceMax must be >= 0", ErrInvalidData)
	}
	if in.PriceMin != nil && in.PriceMax != nil && *in.PriceMin > *in.PriceMax {
		return infrastructure.ListSessionsParams{}, fmt.Errorf("%w: priceMin must be <= priceMax", ErrInvalidData)
	}
	if in.DateFrom != nil && in.DateTo != nil && !in.DateFrom.Before(*in.DateTo) {
		return infrastructure.ListSessionsParams{}, fmt.Errorf("%w: dateFrom must be < dateTo", ErrInvalidData)
	}

	// --- limit ---------------------------------------------------------
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	// --- status filter --------------
	statuses := resolveStatusFilter(in.Status, scope, targetIsViewer)

	return infrastructure.ListSessionsParams{
		Viewer:         in.Viewer,
		Scope:          scope,
		MasterID:       masterID,
		PlayerID:       playerID,
		Status:         statuses,
		TargetIsViewer: targetIsViewer,
		Search:         in.Search,
		Format:         format,
		Type:           stype,
		City:           in.City,
		SystemID:       in.SystemID,
		HasFreeSeats:   in.HasFreeSeats,
		PriceMin:       in.PriceMin,
		PriceMax:       in.PriceMax,
		DateFrom:       in.DateFrom,
		DateTo:         in.DateTo,
		Sort:           sort,
		SortOrder:      order,
		Cursor:         in.Cursor,
		Limit:          limit,
	}, nil
}

func isValidSessionSort(s dtos.SessionListSort) bool {
	switch s {
	case dtos.SortSessionScheduledAt,
		dtos.SortSessionCreatedAt,
		dtos.SortSessionPrice,
		dtos.SortSessionTitle,
		dtos.SortSessionSystem:
		return true
	}
	return false
}
