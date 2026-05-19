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
	Sort        string     `json:"s"`
	SortOrder   string     `json:"o"` // "asc" or "desc"
	ScheduledAt *time.Time `json:"sa,omitempty"`
	CreatedAt   *time.Time `json:"c,omitempty"`
	Price       *float64   `json:"p,omitempty"`
	Title       *string    `json:"t,omitempty"`
	ID          string     `json:"id"`
}

func orderString(asc bool) string {
	if asc {
		return "asc"
	}
	return "desc"
}

// applyCursor adds the keyset-pagination WHERE clause to q based on the
// previous page's cursor. nil cursor → unchanged q. Sort/order mismatch
// between cursor and current request → cursor.ErrInvalidCursor.
func applyCursor(q sq.SelectBuilder, c *sessionCursor, sortKey string, asc bool) (sq.SelectBuilder, error) {
	if c == nil {
		return q, nil
	}
	if c.Sort != sortKey || c.SortOrder != orderString(asc) {
		return q, cursor.ErrInvalidCursor
	}

	sortCol, ok := sortColumns[sortKey]
	if !ok {
		return q, cursor.ErrInvalidCursor
	}

	op := "<"
	if asc {
		op = ">"
	}

	switch sortKey {
	case "scheduled_at":
		if c.ScheduledAt == nil {
			return q, cursor.ErrInvalidCursor
		}
		return q.Where(fmt.Sprintf("(%s, s.id) %s (?, ?)", sortCol, op), *c.ScheduledAt, c.ID), nil
	case "created_at":
		if c.CreatedAt == nil {
			return q, cursor.ErrInvalidCursor
		}
		return q.Where(fmt.Sprintf("(%s, s.id) %s (?, ?)", sortCol, op), *c.CreatedAt, c.ID), nil
	case "price":
		if c.Price == nil {
			return q, cursor.ErrInvalidCursor
		}
		return q.Where(fmt.Sprintf("(%s, s.id) %s (?, ?)", sortCol, op), *c.Price, c.ID), nil
	case "title":
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
func buildNextCursor(last dtos.Session, sortKey string, asc bool) (string, error) {
	c := sessionCursor{
		Sort:      sortKey,
		SortOrder: orderString(asc),
		ID:        last.Id,
	}
	switch sortKey {
	case "scheduled_at":
		if last.ScheduledAt == nil {
			return "", errors.New("build next cursor: scheduled_at sort requires non-null ScheduledAt")
		}
		v := *last.ScheduledAt
		c.ScheduledAt = &v
	case "created_at":
		v := last.CreatedAt
		c.CreatedAt = &v
	case "price":
		v := last.Price
		c.Price = &v
	case "title":
		v := last.Title
		c.Title = &v
	}
	return cursor.EncodeCursor(c)
}
