package infrastructure

import (
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/itzLilix/questboard-shared/cursor"
	"github.com/itzLilix/questboard-shared/dtos"
)

// --- session-specific cursor ------------------------------------------------

// sessionCursor carries the sort state + position needed to continue a page.
// Sort + SortOrder are validated against the current request at applyCursor
// time so paginating with a different sort produces cursor.ErrInvalidCursor.
type sessionCursor struct {
	Sort        dtos.SessionListSort     `json:"s"`
	SortOrder   dtos.SortOrder     `json:"o"` 
	ScheduledAt *time.Time `json:"sa,omitempty"`
	CreatedAt   *time.Time `json:"c,omitempty"`
	Price       *float64   `json:"p,omitempty"`
	Title       *string    `json:"t,omitempty"`
	System      *string    `json:"sy,omitempty"`
	ID          string     `json:"id"`
}

// applyCursor adds the keyset-pagination WHERE clause to q based on the
// previous page's cursor. nil cursor → unchanged q. Sort/order mismatch
// between cursor and current request → cursor.ErrInvalidCursor.
func applyCursor(q sq.SelectBuilder, c *sessionCursor, sortKey dtos.SessionListSort, sortOrder dtos.SortOrder) (sq.SelectBuilder, error) {
	if c == nil {
		return q, nil
	}
	if c.Sort != sortKey || c.SortOrder != sortOrder {
		return q, cursor.ErrInvalidCursor
	}

	sortCol, ok := sortColumns[sortKey]
	if !ok {
		return q, cursor.ErrInvalidCursor
	}

	op := "<"
	if sortOrder == dtos.SortAsc {
		op = ">"
	}

	switch sortKey {
	case dtos.SortSessionScheduledAt:
		if c.ScheduledAt == nil {
			return q, cursor.ErrInvalidCursor
		}
		return q.Where(fmt.Sprintf("(%s, s.id) %s (?, ?)", sortCol, op), *c.ScheduledAt, c.ID), nil
	case dtos.SortSessionCreatedAt:
		if c.CreatedAt == nil {
			return q, cursor.ErrInvalidCursor
		}
		return q.Where(fmt.Sprintf("(%s, s.id) %s (?, ?)", sortCol, op), *c.CreatedAt, c.ID), nil
	case dtos.SortSessionPrice:
		if c.Price == nil {
			return q, cursor.ErrInvalidCursor
		}
		return q.Where(fmt.Sprintf("(%s, s.id) %s (?, ?)", sortCol, op), *c.Price, c.ID), nil
	case dtos.SortSessionTitle:
		if c.Title == nil {
			return q, cursor.ErrInvalidCursor
		}
		return q.Where(fmt.Sprintf("(%s, s.id) %s (?, ?)", sortCol, op), *c.Title, c.ID), nil
	}
	return q.Where(fmt.Sprintf("s.id %s ?", op), c.ID), nil
}

// buildNextCursor builds the cursor string from the last visible row of the
// current page. Returns "" when last is the empty value (caller should not
// have called it in that case).
func buildNextCursor(last dtos.Session, sortKey dtos.SessionListSort, sortOrder dtos.SortOrder) (string, error) {
	c := sessionCursor{
		Sort:      sortKey,
		SortOrder: sortOrder,
		ID:        last.Id,
	}
	switch sortKey {
	case dtos.SortSessionScheduledAt:
		if last.ScheduledAt == nil {
			return "", errors.New("build next cursor: scheduled_at sort requires non-null ScheduledAt")
		}
		v := *last.ScheduledAt
		c.ScheduledAt = &v
	case dtos.SortSessionCreatedAt:
		v := last.CreatedAt
		c.CreatedAt = &v
	case dtos.SortSessionPrice:
		v := last.Price
		c.Price = &v
	case dtos.SortSessionTitle:
		v := last.Title
		c.Title = &v
	case dtos.SortSessionSystem:
		v := last.System.Name
		c.System = &v
	}
	return cursor.EncodeCursor(c)
}
