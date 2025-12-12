package googleapi

import (
	"context"

	"google.golang.org/api/calendar/v3"

	"github.com/steipete/gogcli/internal/googleauth"
)

func NewCalendar(ctx context.Context, email string) (*calendar.Service, error) {
	opts, err := optionsForAccount(ctx, googleauth.ServiceCalendar, email)
	if err != nil {
		return nil, err
	}
	return calendar.NewService(ctx, opts...)
}
