# classroom.md

> Workspace-only: Classroom commands require Google Workspace for Education/eligible domain setup.

High-surface service (70+ commands): courses, roster, coursework, materials, submissions, announcements, topics, invitations, guardians.

## Courses
- `wk classroom courses list|get|create|update|archive|unarchive|join|leave|delete|url`

## Roster: students + teachers
- `wk classroom students list|get|add|remove`
- `wk classroom teachers list|get|add|remove`
- `wk classroom roster <courseId>`

## Coursework + materials + topics
- `wk classroom coursework list|get|create|update|delete|assignees`
- `wk classroom materials list|get|create|update|delete`
- `wk classroom topics list|get|create|update|delete`

## Submissions lifecycle
- `wk classroom submissions list|get`
- `wk classroom submissions turn-in|reclaim|return`
- `wk classroom submissions grade <courseId> <courseWorkId> <submissionId> --draft-grade <n>`

## Announcements
- `wk classroom announcements list|get|create|update|delete|assignees`

## Invitations, guardians, profiles
- `wk classroom invitations list|get|create|accept|delete`
- `wk classroom guardians list|get|delete` (args: `<studentId> [<guardianId>]`)
- `wk classroom guardian-invitations list|get|create` (args: `<studentId> [<invitationId>]`)
- `wk classroom profile get [<userId>]` (default: me)

## Example
```bash
wk classroom courses list --read-only
wk classroom coursework create <courseId> --title 'Homework 5' --dry-run
wk classroom submissions grade <courseId> <courseWorkId> <submissionId> --draft-grade 95 --dry-run
```
