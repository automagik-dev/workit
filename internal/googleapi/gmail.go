package googleapi

import (
	"context"

	"google.golang.org/api/gmail/v1"

	"github.com/steipete/gogcli/internal/googleauth"
)

func NewGmail(ctx context.Context, email string) (*gmail.Service, error) {
	opts, err := optionsForAccount(ctx, googleauth.ServiceGmail, email)
	if err != nil {
		return nil, err
	}
	return gmail.NewService(ctx, opts...)
}
