package googleapi

import (
	"context"

	"google.golang.org/api/people/v1"

	"github.com/steipete/gogcli/internal/googleauth"
)

func NewPeople(ctx context.Context, email string) (*people.Service, error) {
	opts, err := optionsForAccount(ctx, googleauth.ServiceContacts, email)
	if err != nil {
		return nil, err
	}
	return people.NewService(ctx, opts...)
}
