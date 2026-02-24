package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	peopleapi "google.golang.org/api/people/v1"

	"github.com/namastexlabs/gog-cli/internal/outfmt"
	"github.com/namastexlabs/gog-cli/internal/ui"
)

// ContactsBatchCmd groups batch contact operations.
type ContactsBatchCmd struct {
	Create ContactsBatchCreateCmd `cmd:"" name:"create" help:"Batch create contacts from JSON"`
	Delete ContactsBatchDeleteCmd `cmd:"" name:"delete" help:"Batch delete contacts"`
}

// ContactsBatchCreateCmd creates contacts in batch from JSON input.
type ContactsBatchCreateCmd struct {
	File string `name:"file" help:"JSON file with contact array (or - for stdin)" default:"-"`
}

// ContactInput represents the simplified input format for batch create.
type ContactInput struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Org        string `json:"organization,omitempty"`
	Title      string `json:"title,omitempty"`
}

// batchCreateResult tracks the outcome of creating a single contact.
type batchCreateResult struct {
	Index  int    `json:"index"`
	Status string `json:"status"`
	Name   string `json:"resourceName,omitempty"`
	Error  string `json:"error,omitempty"`
}

const (
	batchCreateChunkSize = 200      // People API limit per batch create request.
	batchDeleteChunkSize = 500      // People API limit per batch delete request.
	maxBatchInputSize    = 10 << 20 // 10 MB max for batch JSON input.
	statusError          = "error"
)

// parseContactInputs reads and validates JSON contact input from a reader.
// Input is limited to maxBatchInputSize to prevent OOM on large payloads.
func parseContactInputs(r io.Reader) ([]ContactInput, error) {
	lr := io.LimitReader(r, maxBatchInputSize+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("read contacts input: %w", err)
	}
	if int64(len(data)) > maxBatchInputSize {
		return nil, fmt.Errorf("contacts input too large (max %d MB)", maxBatchInputSize>>20)
	}
	var contacts []ContactInput
	if err := json.Unmarshal(data, &contacts); err != nil {
		return nil, fmt.Errorf("parse contacts JSON: %w", err)
	}
	if len(contacts) == 0 {
		return nil, usage("no contacts provided")
	}
	return contacts, nil
}

// parseResourceNames reads JSON array of resource name strings from a reader.
// Input is limited to maxBatchInputSize to prevent OOM on large payloads.
func parseResourceNames(r io.Reader) ([]string, error) {
	lr := io.LimitReader(r, maxBatchInputSize+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("read resource names input: %w", err)
	}
	if int64(len(data)) > maxBatchInputSize {
		return nil, fmt.Errorf("resource names input too large (max %d MB)", maxBatchInputSize>>20)
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return nil, fmt.Errorf("parse resource names: %w", err)
	}
	return names, nil
}

func (c *ContactsBatchCreateCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	// Read input.
	var reader io.Reader
	if c.File == "-" {
		reader = os.Stdin
	} else {
		f, err := os.Open(c.File)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer f.Close()
		reader = f
	}

	contacts, err := parseContactInputs(reader)
	if err != nil {
		return err
	}

	if dryErr := dryRunExit(ctx, flags, "contacts.batch.create", map[string]any{
		"count": len(contacts),
	}); dryErr != nil {
		return dryErr
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	svc, err := newPeopleContactsService(ctx, account)
	if err != nil {
		return err
	}

	var results []batchCreateResult

	// Process in chunks using native batch API.
	for i := 0; i < len(contacts); i += batchCreateChunkSize {
		end := i + batchCreateChunkSize
		if end > len(contacts) {
			end = len(contacts)
		}
		chunk := contacts[i:end]

		contactsToCreate := make([]*peopleapi.ContactToCreate, 0, len(chunk))
		for _, ci := range chunk {
			person := &peopleapi.Person{
				Names: []*peopleapi.Name{{
					GivenName:  ci.GivenName,
					FamilyName: ci.FamilyName,
				}},
			}
			if ci.Email != "" {
				person.EmailAddresses = []*peopleapi.EmailAddress{{Value: ci.Email}}
			}
			if ci.Phone != "" {
				person.PhoneNumbers = []*peopleapi.PhoneNumber{{Value: ci.Phone}}
			}
			if ci.Org != "" {
				person.Organizations = []*peopleapi.Organization{{Name: ci.Org, Title: ci.Title}}
			}
			contactsToCreate = append(contactsToCreate, &peopleapi.ContactToCreate{
				ContactPerson: person,
			})
		}

		resp, batchErr := svc.People.BatchCreateContacts(&peopleapi.BatchCreateContactsRequest{
			Contacts: contactsToCreate,
			ReadMask: "names",
		}).Context(ctx).Do()

		if batchErr != nil {
			// If the entire batch fails, mark all contacts in this chunk as errors.
			for j := range chunk {
				results = append(results, batchCreateResult{
					Index:  i + j,
					Status: statusError,
					Error:  batchErr.Error(),
				})
			}
			continue
		}

		// Map results from batch response.
		for j := range chunk {
			r := batchCreateResult{Index: i + j, Status: "created"}
			if resp == nil || j >= len(resp.CreatedPeople) || resp.CreatedPeople[j] == nil {
				// Response entry missing for this contact -- cannot confirm creation.
				r.Status = statusError
				r.Error = "no response entry from API"
				results = append(results, r)
				continue
			}
			cp := resp.CreatedPeople[j]
			if cp.Person != nil {
				r.Name = cp.Person.ResourceName
			}
			if cp.Status != nil && cp.Status.Code != 0 {
				r.Status = statusError
				r.Error = cp.Status.Message
			}
			results = append(results, r)
		}
	}

	created := 0
	errCount := 0
	for _, r := range results {
		if r.Status == "created" {
			created++
		} else {
			errCount++
		}
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"results": results,
			"total":   len(contacts),
			"created": created,
			"errors":  errCount,
		})
	}

	for _, r := range results {
		if r.Status == "created" {
			u.Out().Printf("created\t%d\t%s", r.Index, r.Name)
		} else {
			u.Err().Printf("error\t%d\t%s", r.Index, r.Error)
		}
	}
	return nil
}

// ContactsBatchDeleteCmd deletes contacts in batch.
type ContactsBatchDeleteCmd struct {
	ResourceNames []string `arg:"" name:"resourceName" help:"Contact resource names to delete (people/...)" optional:""`
	File          string   `name:"file" help:"JSON file with resource name array (or - for stdin)"`
}

func (c *ContactsBatchDeleteCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	names := append([]string(nil), c.ResourceNames...)

	// If --file provided, read from file/stdin instead.
	if c.File != "" {
		var reader io.Reader
		if c.File == "-" {
			reader = os.Stdin
		} else {
			f, err := os.Open(c.File)
			if err != nil {
				return fmt.Errorf("open file: %w", err)
			}
			defer f.Close()
			reader = f
		}
		fileNames, err := parseResourceNames(reader)
		if err != nil {
			return err
		}
		names = append(names, fileNames...)
	}

	// Filter empty names.
	filtered := names[:0]
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			filtered = append(filtered, n)
		}
	}
	names = filtered

	if len(names) == 0 {
		return usage("required: resource names (positional args or --file)")
	}

	if err := dryRunExit(ctx, flags, "contacts.batch.delete", map[string]any{
		"count": len(names),
		"names": names,
	}); err != nil {
		return err
	}

	if confirmErr := confirmDestructive(ctx, flags, fmt.Sprintf("batch delete %d contacts", len(names))); confirmErr != nil {
		return confirmErr
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	svc, err := newPeopleContactsService(ctx, account)
	if err != nil {
		return err
	}

	// Process in chunks using native batch API.
	type deleteChunkResult struct {
		ChunkStart int      `json:"chunkStart"`
		Count      int      `json:"count"`
		Status     string   `json:"status"`
		Names      []string `json:"names"`
		Error      string   `json:"error,omitempty"`
	}

	var results []deleteChunkResult
	totalDeleted := 0
	totalErrors := 0

	for i := 0; i < len(names); i += batchDeleteChunkSize {
		end := i + batchDeleteChunkSize
		if end > len(names) {
			end = len(names)
		}
		chunk := names[i:end]

		_, batchErr := svc.People.BatchDeleteContacts(&peopleapi.BatchDeleteContactsRequest{
			ResourceNames: chunk,
		}).Context(ctx).Do()

		r := deleteChunkResult{
			ChunkStart: i,
			Count:      len(chunk),
			Names:      chunk,
			Status:     "deleted",
		}
		if batchErr != nil {
			r.Status = statusError
			r.Error = batchErr.Error()
			totalErrors += len(chunk)
		} else {
			totalDeleted += len(chunk)
		}
		results = append(results, r)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"results": results,
			"total":   len(names),
			"deleted": totalDeleted,
			"errors":  totalErrors,
		})
	}

	for _, r := range results {
		if r.Status == "deleted" {
			for _, name := range r.Names {
				u.Out().Printf("deleted\t%s", name)
			}
		} else {
			u.Err().Printf("error\t%d contacts\t%s", r.Count, r.Error)
		}
	}
	return nil
}
