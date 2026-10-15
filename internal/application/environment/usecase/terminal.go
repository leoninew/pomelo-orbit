package environmentsvc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"time"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	TerminalTicketLifetime = 30 * time.Second
	terminalIdleLimit      = 15 * time.Minute
	TerminalSessionLimit   = 2 * time.Hour
	TerminalConnectLimit   = 15 * time.Second
	terminalCheckInterval  = 2 * time.Second
	maxTerminalTickets     = 1024
	maxTerminalSessions    = 16
	maxUserTerminals       = 2
)

type terminalTicket struct {
	actorId            string
	projectId          string
	environmentId      string
	targetType         string
	targetRevision     int64
	credentialId       string
	credentialRevision int64
	expiresAt          time.Time
}

type TerminalGrant struct {
	ActorId   string
	ProjectId string
	Target    environmentport.Target
}

type TerminalService struct {
	environments Service
	mu           sync.Mutex
	tickets      map[[32]byte]terminalTicket
	active       map[string]int
	activeTotal  int
}

func NewTerminalService(environments Service) *TerminalService {
	return &TerminalService{environments: environments,
		tickets: make(map[[32]byte]terminalTicket), active: make(map[string]int)}
}

func (s *TerminalService) IssueTicket(ctx context.Context, actorId, projectId string) (string, error) {
	if err := s.environments.ensureProjectMembership(ctx, projectId, actorId); err != nil {
		return "", err
	}
	target, err := s.environments.resolveConnectionTarget(ctx, projectId)
	if err != nil {
		return "", err
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create terminal ticket", err)
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, ticket := range s.tickets {
		if !now.Before(ticket.expiresAt) {
			delete(s.tickets, key)
		}
	}
	if len(s.tickets) >= maxTerminalTickets {
		return "", apperror.New(apperror.KindUnavailable, "Terminal ticket capacity reached")
	}
	ticket := terminalTargetIdentity(target.Environment)
	ticket.actorId, ticket.projectId, ticket.expiresAt = actorId, projectId, now.Add(TerminalTicketLifetime)
	s.tickets[sha256.Sum256(secret[:])] = ticket
	return base64.RawURLEncoding.EncodeToString(secret[:]), nil
}

func (s *TerminalService) RedeemTicket(ctx context.Context, projectId, token string) (TerminalGrant, func(), error) {
	secret, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(secret) != 32 {
		return TerminalGrant{}, nil, apperror.New(apperror.KindUnauthorized, "Invalid terminal ticket")
	}
	key := sha256.Sum256(secret)
	s.mu.Lock()
	ticket, found := s.tickets[key]
	delete(s.tickets, key)
	s.mu.Unlock()
	if !found || !time.Now().Before(ticket.expiresAt) || ticket.projectId != projectId {
		return TerminalGrant{}, nil, apperror.New(apperror.KindUnauthorized, "Invalid terminal ticket")
	}
	if err := s.environments.ensureProjectMembership(ctx, projectId, ticket.actorId); err != nil {
		return TerminalGrant{}, nil, err
	}
	target, err := s.environments.resolveConnectionTarget(ctx, projectId)
	if err != nil {
		return TerminalGrant{}, nil, err
	}
	if !matchesTerminalTarget(ticket, target.Environment) {
		return TerminalGrant{}, nil, apperror.New(apperror.KindConflict, "Environment changed after terminal ticket was issued")
	}
	return s.acquire(ticket.actorId, projectId, target)
}

func (s *TerminalService) Acquire(ctx context.Context, actorId, projectId string) (TerminalGrant, func(), error) {
	if err := s.environments.ensureProjectMembership(ctx, projectId, actorId); err != nil {
		return TerminalGrant{}, nil, err
	}
	target, err := s.environments.resolveConnectionTarget(ctx, projectId)
	if err != nil {
		return TerminalGrant{}, nil, err
	}
	return s.acquire(actorId, projectId, target)
}

func (s *TerminalService) acquire(actorId, projectId string, target environmentport.Target) (TerminalGrant, func(), error) {
	s.mu.Lock()
	if s.activeTotal >= maxTerminalSessions || s.active[actorId] >= maxUserTerminals {
		s.mu.Unlock()
		return TerminalGrant{}, nil, apperror.New(apperror.KindUnavailable, "Terminal session capacity reached")
	}
	s.activeTotal++
	s.active[actorId]++
	s.mu.Unlock()
	var once sync.Once
	release := func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.activeTotal--
			s.active[actorId]--
			if s.active[actorId] == 0 {
				delete(s.active, actorId)
			}
		})
	}
	return TerminalGrant{ActorId: actorId, ProjectId: projectId, Target: target}, release, nil
}

func (s *TerminalService) ValidateSession(ctx context.Context, grant TerminalGrant) error {
	return s.environments.validateConnection(ctx, grant)
}

func (s Service) validateConnection(ctx context.Context, grant TerminalGrant) error {
	if err := s.ensureProjectMembership(ctx, grant.ProjectId, grant.ActorId); err != nil {
		return err
	}
	current, err := s.environmentForProject(ctx, grant.ProjectId)
	if err != nil {
		return err
	}
	if !matchesTerminalTarget(terminalTargetIdentity(grant.Target.Environment), current) {
		return apperror.New(apperror.KindConflict, "Project environment changed during connection")
	}
	if current.IsLocal() {
		return nil
	}
	if expected := grant.Target.Environment.SSH.HostKeyFingerprint; expected != "" && current.SSH.HostKeyFingerprint != expected {
		return apperror.New(apperror.KindConflict, "Project SSH host key changed during connection")
	}
	credential, err := s.environmentCredential(ctx, current.SSH.CredentialId)
	if err != nil || !matchesEnvironmentCredential(current, credential) {
		return apperror.New(apperror.KindConflict, "Project SSH credential changed during connection")
	}
	return nil
}

func matchesTerminalTarget(ticket terminalTicket, current model.Environment) bool {
	if current.Id != ticket.environmentId || current.TargetType != ticket.targetType || current.TargetRevision != ticket.targetRevision {
		return false
	}
	return current.IsLocal() || (current.IsSSH() && current.SSH.CredentialId == ticket.credentialId &&
		current.SSH.CredentialRevision == ticket.credentialRevision)
}

func terminalTargetIdentity(environment model.Environment) terminalTicket {
	ticket := terminalTicket{environmentId: environment.Id, targetType: environment.TargetType, targetRevision: environment.TargetRevision}
	if environment.IsSSH() {
		ticket.credentialId, ticket.credentialRevision = environment.SSH.CredentialId, environment.SSH.CredentialRevision
	}
	return ticket
}

func (s Service) resolveConnectionTarget(ctx context.Context, projectId string) (environmentport.Target, error) {
	item, err := s.environmentForProject(ctx, projectId)
	if err != nil {
		return environmentport.Target{}, err
	}
	if err := validateEnvironment(item, s.localDisplay.Platform, false, false); err != nil {
		return environmentport.Target{}, err
	}
	target := environmentport.Target{Environment: item}
	if item.IsLocal() {
		return target, nil
	}
	credential, err := s.environmentCredential(ctx, item.SSH.CredentialId)
	if err != nil || !matchesEnvironmentCredential(item, credential) {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Environment SSH credential binding is invalid")
	}
	privateKey, err := decryptEnvironmentPrivateKey(s.secretKey, credential)
	if err != nil {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Environment SSH credential binding is invalid")
	}
	target.PrivateKey = &privateKey
	return target, nil
}

// ConfirmConnection pins the key observed on the authenticated terminal connection
// before the handler exposes output or accepts browser input.
func (s *TerminalService) ConfirmConnection(ctx context.Context, grant TerminalGrant, fingerprint string) (TerminalGrant, error) {
	return s.environments.confirmConnection(ctx, grant, fingerprint)
}

func (s Service) confirmConnection(ctx context.Context, grant TerminalGrant, fingerprint string) (TerminalGrant, error) {
	if err := s.validateConnection(ctx, grant); err != nil {
		return TerminalGrant{}, err
	}
	if grant.Target.Environment.IsLocal() {
		return grant, nil
	}
	if err := s.recordHostKey(ctx, grant.Target.Environment, fingerprint); err != nil {
		return TerminalGrant{}, err
	}
	ssh := *grant.Target.Environment.SSH
	ssh.HostKeyFingerprint = fingerprint
	grant.Target.Environment.SSH = &ssh
	if err := s.validateConnection(ctx, grant); err != nil {
		return TerminalGrant{}, err
	}
	return grant, nil
}

func (s *TerminalService) RuntimeUsername(grant TerminalGrant) string {
	if grant.Target.Environment.IsLocal() {
		return s.environments.localDisplay.Username
	}
	return grant.Target.Environment.SSH.Username
}
