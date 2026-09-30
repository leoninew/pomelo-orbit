package environmentsvc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	TerminalTicketLifetime = 30 * time.Second
	TerminalIdleLimit      = 15 * time.Minute
	TerminalSessionLimit   = 2 * time.Hour
	TerminalConnectLimit   = 15 * time.Second
	TerminalCheckInterval  = 2 * time.Second
	maxTerminalTickets     = 1024
	maxTerminalSessions    = 16
	maxUserTerminals       = 2
)

type terminalTicket struct {
	actorId            string
	projectId          string
	environmentId      string
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
	resolver     environmentport.TargetResolver
	mu           sync.Mutex
	tickets      map[[32]byte]terminalTicket
	active       map[string]int
	activeTotal  int
}

func NewTerminalService(environments Service, resolver environmentport.TargetResolver) *TerminalService {
	return &TerminalService{environments: environments, resolver: resolver,
		tickets: make(map[[32]byte]terminalTicket), active: make(map[string]int)}
}

func (s *TerminalService) IssueTicket(ctx context.Context, actorId, projectId string) (string, error) {
	if err := s.environments.ensureProjectMembership(ctx, projectId, actorId); err != nil {
		return "", err
	}
	target, err := s.resolver.ResolveProjectTarget(ctx, projectId)
	if err != nil {
		return "", err
	}
	if !target.Environment.IsSSH() || target.PrivateKey == nil {
		return "", apperror.New(apperror.KindValidation, "Terminal requires a ready SSH environment")
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
	s.tickets[sha256.Sum256(secret[:])] = terminalTicket{
		actorId: actorId, projectId: projectId, environmentId: target.Environment.Id,
		targetRevision: target.Environment.TargetRevision, credentialId: target.Environment.SSH.CredentialId,
		credentialRevision: target.Environment.SSH.CredentialRevision, expiresAt: now.Add(TerminalTicketLifetime),
	}
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
	target, err := s.resolver.ResolveProjectTarget(ctx, projectId)
	if err != nil {
		return TerminalGrant{}, nil, err
	}
	if !matchesTerminalTarget(ticket, target.Environment) {
		return TerminalGrant{}, nil, apperror.New(apperror.KindConflict, "Environment changed after terminal ticket was issued")
	}
	s.mu.Lock()
	if s.activeTotal >= maxTerminalSessions || s.active[ticket.actorId] >= maxUserTerminals {
		s.mu.Unlock()
		return TerminalGrant{}, nil, apperror.New(apperror.KindUnavailable, "Terminal session capacity reached")
	}
	s.activeTotal++
	s.active[ticket.actorId]++
	s.mu.Unlock()
	var once sync.Once
	release := func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.activeTotal--
			s.active[ticket.actorId]--
			if s.active[ticket.actorId] == 0 {
				delete(s.active, ticket.actorId)
			}
		})
	}
	return TerminalGrant{ActorId: ticket.actorId, ProjectId: projectId, Target: target}, release, nil
}

func (s *TerminalService) ValidateSession(ctx context.Context, grant TerminalGrant) error {
	if err := s.environments.ensureProjectMembership(ctx, grant.ProjectId, grant.ActorId); err != nil {
		return err
	}
	current, err := s.environments.environmentForProject(ctx, grant.ProjectId)
	if err != nil {
		return err
	}
	if !current.HasFreshSuccessfulProbe() || !matchesTerminalTarget(terminalTicket{
		environmentId: grant.Target.Environment.Id, targetRevision: grant.Target.Environment.TargetRevision,
		credentialId: grant.Target.Environment.SSH.CredentialId, credentialRevision: grant.Target.Environment.SSH.CredentialRevision,
	}, current) {
		return errors.New("project SSH environment changed during terminal session")
	}
	credential, err := s.environments.environmentCredentials.EnvironmentCredential(ctx, current.SSH.CredentialId)
	if err != nil || !matchesEnvironmentCredential(current, credential) {
		return errors.New("project SSH credential changed during terminal session")
	}
	return nil
}

func matchesTerminalTarget(ticket terminalTicket, current model.Environment) bool {
	return current.IsSSH() && current.Id == ticket.environmentId &&
		current.TargetRevision == ticket.targetRevision && current.SSH.CredentialId == ticket.credentialId &&
		current.SSH.CredentialRevision == ticket.credentialRevision && current.HasFreshSuccessfulProbe()
}
