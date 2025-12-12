package googleapi

import (
	"context"

	"google.golang.org/api/drive/v3"

	"github.com/steipete/gogcli/internal/googleauth"
)

func NewDrive(ctx context.Context, email string) (*drive.Service, error) {
	opts, err := optionsForAccount(ctx, googleauth.ServiceDrive, email)
	if err != nil {
		return nil, err
	}
	return drive.NewService(ctx, opts...)
}
