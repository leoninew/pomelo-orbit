package applicationsvc

import (
	"context"
	"errors"
	"strings"

	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func (s Service) ImportApplication(ctx context.Context, userId string, input applicationdto.ApplicationImportInput) (model.Application, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return model.Application{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Application{}, err
	}
	name, code, kind, err := normalizeApplicationImportInput(input)
	if err != nil {
		return model.Application{}, err
	}
	if err := s.ensureApplicationNameAvailable(ctx, name); err != nil {
		return model.Application{}, err
	}
	if err := s.ensureApplicationCodeAvailable(ctx, code); err != nil {
		return model.Application{}, err
	}
	app := model.Application{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Code: code, Kind: kind}
	if err := s.store.CreateApplication(ctx, app); err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to import application", err)
	}
	label := strings.TrimSpace(input.VersionLabel)
	if label == "" {
		label = "v1"
	}
	if _, err := s.CreateVersion(ctx, userId, applicationdto.VersionCreateInput{
		ApplicationId: app.Id, Label: label, Note: input.VersionNote,
		Components: input.Components,
	}); err != nil {
		_ = s.store.DeleteApplication(ctx, app.Id)
		return model.Application{}, err
	}
	created, err := s.store.Application(ctx, app.Id)
	if err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return created, nil
}

func (s Service) ExportApplication(ctx context.Context, userId string, applicationId string) (applicationdto.ApplicationExport, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return applicationdto.ApplicationExport{}, err
	}
	versions, err := s.ListVersions(ctx, userId, app.Id)
	if err != nil {
		return applicationdto.ApplicationExport{}, err
	}
	return applicationdto.ApplicationExport{Application: app, Versions: versions}, nil
}

func normalizeApplicationImportInput(input applicationdto.ApplicationImportInput) (string, string, string, error) {
	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	kind, err := normalizeApplicationKind(input.Kind)
	if err != nil {
		return "", "", "", err
	}
	if name == "" || len(name) > 100 || code == "" || len(code) > 100 || !applicationCreateCodePattern.MatchString(code) {
		return "", "", "", apperror.New(apperror.KindValidation, "Invalid application fields")
	}
	return name, code, kind, nil
}

func (s Service) ensureApplicationCodeAvailable(ctx context.Context, code string) error {
	existing, err := s.store.ApplicationByCode(ctx, code)
	if err == nil {
		return apperror.New(apperror.KindValidation, "Application code '"+existing.Code+"' already exists")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check application code", err)
	}
	return nil
}

func optionalText(value *string) *string {
	if value == nil {
		return nil
	}
	text := *value
	return &text
}
