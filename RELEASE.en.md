# Release Notes

This file records the business capabilities of `develop` in reverse chronological order, with one section per date that has commits.

## 2026-11-16

### Host identity policy

- SSH environments gain two policies: "negotiated identity" and "stable identity". Stable identity pins the target by its host public key, so a reissued host certificate with the same key still connects; the default keeps pinning the negotiated fingerprint.
- Switching policies clears the pinned fingerprint, increments the target revision, and keeps existing credentials; re-probing and verifying the fingerprint are required afterwards. Environment details show the current policy and pinned value.
- Project handover packages use format `3` and restore the identity policy when overwriting environments; the `000057` migration leaves existing environments unchanged. Before downgrading with stable identity, switch back to negotiated identity and record the fingerprint again.

### Fixes

- Service status-check failures now include the execution environment and target type, plus the actual Docker query command, making host, directory, and container problems easier to tell apart.
- Deployment-dialogue messages show local times and can be copied; history timestamps are saved with the dialogue.
- Markdown content is sanitized with the browser DOMPurify instance, removing unsafe HTML.
- The pipeline filter area scrolls while the create action stays aligned.

## 2026-11-11

### Development environment

- JavaScript workspaces move from Yarn 4 to a pinned pnpm version, and local, CI, Docker, and documentation use the same Node.js and Corepack versions. Source and release archive layouts are unchanged.

## 2026-11-09

### Service controlled files

- A service can set content, permissions, and write policy per controlled file, while directories and regular files remain read-only.
- Deployment can overwrite controlled files or only fill in missing ones while keeping existing content; application-managed files are no longer unconditionally overwritten on recreation.
- The service editor, version overrides, and MCP synchronization support this configuration, and HTTP and MCP can read the full declaration.

## 2026-11-06

### Cross-project service copy

- Service details gain "Copy service", which copies applications, versions, components, and service configuration to another project the user can access, with new IDs and internal references.
- The latest version is copied by default, or all versions; the target deployment directory is derived from the target environment and remains editable.
- Copying follows project permissions and rolls back entirely on code conflicts; the target project opens with the new service afterwards, without deploying it.

### Fixes

- Service status filters persist per project, and deployment filters reset on project changes instead of carrying over the previous project's conditions.

## 2026-11-04

### Operations tooling

- A new Orbit database merge script combines the business data of a chosen project across two PostgreSQL databases and writes the result to both: the newest configuration wins per object, other records are unioned, and primary keys and references are unified.
- It supports read-only plan generation, review of changes and ID mapping before execution, resumable runs, and result verification. Encrypted credentials require the Fernet keys of both instances.

## 2026-11-03

### Workspace navigation

- Account actions move to the sidebar footer, and environment commands such as initialization go into the page toolbar, leaving more room for business content.

## 2026-11-02

### User guide

- The handbook is reorganized around installation, first deployment, daily work, and maintenance, and is published on GitHub Pages.
- Two self-hosted deployment paths are added, demonstrating NGINX access through the deployment dialogue with internal port `80` and `nginx.lvh.me`.

## 2026-10-31

### Application details

- Application details use a collapsible version/component tree, with version or component details in a separate panel for the selected node and keyboard navigation.
- The dialogue page unifies message and composer layout, shows message times, and supports copying.

## 2026-10-30

### Cloudflare DNS

- Reusing the environment gateway's DNS API token, the DNS page searches visible zones and manages A records: creation, deletion, and changes to IPv4, TTL, and proxy status.
- Changes go directly to Cloudflare without requiring the gateway container to run; HTTP and MCP provide the same capability with project permissions.

## 2026-10-29

### Access tokens

- Saved access tokens are masked by default and can be revealed and copied from the token list; authentication still uses the token hash.
- Viewable values are encrypted with `POMELO_ORBIT_JWT__SECRET_KEY`; tokens that only stored a hash before the migration cannot be recovered and must be recreated.

### Fixes

- Tables align and ID columns show full values; semantic type sizes are unified, with 12px reserved for code, commands, and logs.
- Credential inputs and search controls use shared components with clearer semantics.

## 2026-10-28

### Interface

- The whole frontend adopts the open-canvas visual language, with adjusted hierarchy on project, status, and detail pages.
- Status, detail, and list layouts align, removing duplicate fallback displays.

## 2026-10-23

### Google login

- Google OAuth login is restored, with state, PKCE, and OIDC checks on the callback; existing accounts are linked by email and new accounts follow the existing rules.

### Fixes

- Routes without valid components are cleaned up, and applied migrations are verified.

## 2026-10-22

### Repository

- The Yarn 4 toolchain is pinned, and legacy documentation is archived so the current handbook is the only usage reference.

## 2026-10-21

### Environment credentials

- SSH environment credentials now belong uniquely to an environment: save the environment first, then initialize credentials explicitly; repository credentials remain separate from execution-host credentials.

## 2026-10-20

### Version and command feedback

- The interface shows the current program version and metadata.
- Operation commands give a clear message in non-interactive terminals instead of exiting silently in scripts.

## 2026-10-19

### Workspace migration

- A new workspace migration tool previews and then relocates data and running Compose services from the old directory, resuming after interruption; it validates the current paths, credentials, and SSH host keys without requiring a fresh probe.

## 2026-10-16

### Routes and environment CLI

- Gateway routes show the managed target identity and access endpoint so the actual destination can be verified.
- Project SSH environments can be listed, opened in an interactive terminal, used to run remote commands, and used for background port forwarding through a local Go program, with access-token authentication and the platform's credentials and host checks; the previous Python CLI and dedicated endpoints are removed.

### Fixes

- Saved deployment directories resolve against the current target after a target change.
- Login sessions and local sessions stay in sync, and secrets in logs are masked.
- SSH key annotations are written, and the Windows initialization flow is corrected.

## 2026-10-15

### Environment CLI

- Project SSH environments support a local CLI authenticated with a personal access token that runs remote commands.

### Fixes

- SSH key annotations explain their purpose, and Windows host configuration is prepared under the current user directory instead of requiring manual path fixes.

## 2026-10-14

### Service status

- Batched service status checks retrieve several services at once, reducing per-item refreshes.

### Pull policy

- Service deployment configures the image pull policy, and filter controls use the same interaction.

## 2026-10-10

### Service operation takeover

- A new deployment can take over an existing managed instance, and the latest accepted operation determines the state; older tasks cannot overwrite newer state.

### Fixes

- Route operation timeouts give clear failure feedback, and route state stays stable after switching projects.

## 2026-10-09

### System settings

- Application configuration overrides saved in system settings persist, and the interface shows the current value source, override value, and settings pending a restart.

### Local terminal

- The environment page gains a local terminal that operates on the execution target from the browser; shared targets show port and directory conflict warnings.

### Fixes

- Colons in run arguments are escaped correctly during script startup.

## 2026-10-08

### Database and scripts

- Database connections are configured with connection URLs.
- The operations CLI has a single entry point, the old database transfer scripts are removed, and backup and restore handle data movement.

## 2026-10-06

### Components and workspace mounts

- Components gain identity markers, and multiple services can share the same workspace mount.
- Service hashes in deployment plans stay consistent, and deployment ownership markers are removed to avoid pointless recreation.

## 2026-10-05

### Fixes

- Log reads no longer require the runtime directory to exist.
- Mount inspection preserves JSON content.
- New gateway components start with a restart policy consistent with the gateway settings.

## 2026-10-04

### Credentials and deployment directories

- Credential updates are tracked and shown more completely in lists.
- Service deployment directories can be confirmed before deployment to prevent path mistakes.
- Container mounts inside Docker host paths are resolved correctly.

## 2026-10-03

### Route syncing

- Route syncing is decoupled from deployment, can run on demand, uses per-stage timeouts, and reports durations and gateway errors; publishing recovers after an environment target change.
- Truncated table text shows its full value in a tooltip.

## 2026-10-02

### SSH deployment

- Remote Compose mount paths resolve against the target host rather than reusing local container paths.

### Gateway

- Traefik API settings are separate from gateway runtime configuration, so changing the API address no longer recreates the container.
- Custom routes persist to the File provider and are restored after the gateway is recreated.

## 2026-10-01

### Fixes

- Deployment and pipeline details state the failure conclusion in a summary first, with details available for expansion.

## 2026-09-30

### Logs

- CI and CD logs share one viewer, and deployments and pipelines stream consistently.

### Environments

- The environment page gains an SSH host terminal that operates on SSH targets from the browser.

## 2026-09-29

### Gateway

- Gateways configure the public site domain and the in-container access address separately, so proxies and business services each use the correct entry point.

### Operations tooling

- Operations scripts gain database copy and backup commands to move data between instances.

## 2026-09-28

### Route certificates

- Certificate settings and route syncing run separately: save certificate configuration first, then apply it during sync.

### Pipelines

- Run details show stage failure reasons and execution details to locate failing stages.

## 2026-09-27

### Routes

- Route lists separate HTTP and TCP entry points with more accurate labels.

## 2026-09-26

### Repositories

- Repositories are shared across projects, and projects reference the same record instead of re-entering addresses and credentials.

## 2026-09-24

### Pipeline templates

- Pipeline templates are global and reusable across projects.
- Templates support version updates, and application pipelines show pending template changes.
- Stage order is required, and template stage defaults appear with the snapshot.

### Fixes

- Windows SSH target home paths are detected correctly, and stage cleanup and SSH logs are no longer lost.
- Project codes are validated by the same rules, and route and gateway pages handle uninitialized state.

## 2026-09-23

### Environment execution

- Pipeline stages run on a project's local or SSH environment, including Windows SSH targets.
- Run snapshots save the execution target, workspace, and variables, and execution restores the target from the snapshot.
- Projects whose environments are not initialized do not proceed to build or deploy.

### Pipelines

- A built-in Docker application pipeline serves as the initial build configuration for new projects.

## 2026-09-22

### Pipelines

- The built-in Docker application pipeline and repository seed keep a runnable example after a database reset.

### Project handover

- Imports reject incomplete routes so handover does not leave broken entry points.

## 2026-09-21

### Pipeline variables

- Variable configuration uses structured, grouped views for inspection and editing.

### Development tools

- Database reset and tunnel diagnostics help restore local environments.

## 2026-09-20

### Pipelines

- Stage variables inherit global defaults, and the variable view is grouped.

### Deployment

- Deployment snapshots keep gateway network configuration so re-creation does not lose the entry network.

## 2026-09-19

### Project handover

- Handover packages are imported only in the current format, with a choice of whether to overwrite the target project environment.
- Imports keep the runtime and route state of transferred services so entry points survive the takeover.

## 2026-09-18

### SSH environments

- SSH initialization generates a command to run on the target host, including Windows.

### Project handover

- Handover states the import identity explicitly, choosing between overwriting an existing project or importing a new one.

### Gateway

- Gateways provide separate site and in-container addresses.

## 2026-09-17

### Project handover

- Project CD resources can be exported and imported into another instance, including applications, versions, services, routes, and deployment records.

### SSH environments

- Probe and execution failures keep diagnostic details that distinguish target-host problems from configuration problems.

## 2026-09-16

### Project isolation

- CI/CD resources belong to projects, and reads and writes enforce project membership permissions.

### Pipelines

- Pipeline templates are decoupled from projects, and multiple projects can instantiate the same template.

### Services

- Service-instance semantics are removed, and services bind directly to application versions.

## 2026-09-15

### SSH environments

- Environments hold their own SSH keys, and initialization commands are generated per environment.
- Credentials are preserved when environments are imported, and Windows targets can be initialized with the command.

## 2026-09-14

### SSH environments

- Windows SSH targets can prepare initialization content locally, and deployment follows probe results.
- Editor assets are served locally for offline use.

## 2026-09-13

### Pipelines

- Pipeline clone workspaces refresh safely before later runs, avoiding stale branches.

## 2026-09-12

### Environment initialization

- A project initialization wizard covers environment probing, gateway preparation, and readiness checks.
- Project workspaces are managed centrally, with roots saved for both local and SSH targets.

### Pipelines

- Pipeline execution targets and workspaces are isolated per project, so projects never share a directory.

## 2026-09-09

### Environments

- A local deployment target lets projects build and deploy without SSH.
- Environment views, probing, and execution are separated, and workspace roots persist per target.

## 2026-09-08

### Environments

- New instances create one usable environment by default, so first use needs no manual baseline setup.

## 2026-09-07

### Pipelines

- Pipelines can be triggered manually for a repository branch or commit, and variables support nested values.
- Stage-level Liquid defaults take effect.

### MCP

- Local MCP access uses access tokens, and tool calls run with the current user's permissions.

## 2026-09-06

### Repositories

- Repositories can be cloned with a Gitea HTTPS token.

## 2026-09-04

### Environments

- Projects can build and deploy through SSH targets.

## 2026-09-03

### Release

- The version comes from `VERSION`, and release archives are produced by the pipeline.

## 2026-09-02

### Authentication

- Users sign in with email, and login redirects no longer nest.

## 2026-09-01

### Database transfer

- PostgreSQL service data can be migrated.

### Dialogue

- Deployment dialogue timeouts accommodate longer model responses and tool-call sequences.

## 2026-08-31

### Components

- Components configure runtime policies such as restart policy, health checks, resources, and runtime parameters, which participate in Compose generation and validation.

## 2026-08-30

### Database

- Platform data runs on SQLite, MySQL, or PostgreSQL.

### Routes

- Custom route syncing is explicit, so deployments do not implicitly rewrite entry points.

## 2026-08-29

### Routes

- Route sync confirms a Traefik snapshot before applying it and can retry based on the result.

### Interface

- Focus returns to the trigger after a dialog closes, keeping keyboard navigation continuous.

## 2026-08-28

### Migrations

- Migration tests cover SQLite and MySQL, so schema results can be verified before upgrades.

## 2026-08-27

### Gateway

- Gateways support DNS-01 certificate issuance, with DNS API tokens stored per environment.
- Gateway schema upgrades work on MySQL.

## 2026-08-26

### Tooling

- Common project commands use a single entry point, reducing differences between local and CI commands.

## 2026-08-24

### Backup

- Instance backups keep Compose files generated during deployment, so restored definitions can be checked directly.

## 2026-08-23

### MCP and plugins

- MCP tools guide the configuration of stateful services.
- Native plugins can be distributed together with release packages.

## 2026-08-22

### Deployment

- Directory, file, controlled-file, and named-volume mount sources must be declared explicitly, and relative paths resolve against the execution target.

### Data migration

- Service data migration uses dbtalk, compatible with more database versions.

## 2026-08-19

### Pipelines

- Stage templates allow duplicate names, so different templates can reuse a name.

### MCP

- Deployment dialogue limits tool-call rounds per turn, preventing unlimited calls in a single round.

### Fixes

- Deployment Compose keeps mount sources, so container mounts are unchanged after recreation.
- Route sync no longer deletes REST routes not managed by the platform.

## 2026-08-18

### Workspaces

- CI/CD workspaces are configurable, and `POMELO_ORBIT_WORKSPACE` centrally adjusts build and deployment paths.

### MCP

- Local MCP access supports browser authorization callbacks, letting signed-in users authorize tool calls directly.

## 2026-08-17

### Deployment

- The service version can be selected before deployment, so a service can switch versions and redeploy.
- Logs rotate daily, and the About dialog shows the current version.

### Data migration

- Service configuration can be exported to an archive and imported into another instance, and database transfer supports SQLite.

### Release

- Direct-run packages and versioned archives are provided, the interface shows the current version, and the log directory defaults under the data directory for easier backups.

## 2026-08-16

### Routes

- Custom route domains default to the gateway base domain, and MCP can read and modify custom routes.

### Environment variables

- Service environment variables can be edited and saved in place.

## 2026-08-15

### Service model

- Services have immutable service codes and their own host directories, and one application can have several services.
- Services can be stopped, and faulted services can be deleted; deleting a service record does not clean up containers or data on the target.
- Applications are decoupled from the gateway runtime, so standard services no longer depend on it.

### Gateway

- Gateways use the same application, version, and service lifecycle as ordinary services.

### Dialogue

- Deployment dialogue keeps session history and can control deployment lifecycle from the conversation.

### CI

- Application pipelines manage variables directly without returning to repository settings.

### Data migration

- A cross-database transfer tool is added, with schema and seed data migrated separately.

## 2026-08-14

### Interface

- Detail pages gain breadcrumbs, and keyword searches run after confirmation.

## 2026-08-13

### Routes

- Custom routes support TCP entry points and access endpoint identity.

## 2026-08-12

### Services

- Service components override runtime parameters, and services and versions store environment variables and runtime configuration separately.

### Pipelines

- Repository reference variables are unified at runtime, no longer distinguishing build and deployment sources.

## 2026-08-11

### Services

- Services can be stopped, and a repository is required only when an application binds its first pipeline.

## 2026-08-10

### Pipelines

- Pipeline template updates can sync to applications, and legacy CI data can be backed up and migrated.

### Services

- Service details show the effective component endpoints instead of requiring comparison with a version.

## 2026-08-09

### Deployment

- Service workspaces are isolated by service code, so services do not share one directory.
- Mount sources can be overridden with host paths.

## 2026-08-08

### Services and routes

- Services gain immutable codes, and component routes derive stable entry points from the service code.

### Async operations

- Deployment, stop, delete, and pipeline execution share the async operation state machine, and the latest accepted operation wins for a service.
- Task states distinguish queued, running, succeeded, failed, and canceled, with readable failure reasons.

### CI

- A reusable stage-template library is added.

### Authentication

- Login sessions restore Fernet CSRF protection with a 3-minute lifetime.

## 2026-08-07

### Application pipelines

- Pipelines are managed per application and can create new versions from build artifacts.

### Components

- Components support entrypoint configuration.

## 2026-08-06

### Deployment dialogue

- A dialogue entry creates and deploys applications, versions, services, and routes through model tool calls within a project; tool calls follow the current user's and project's permissions.
- Conversation records and session history are kept for resuming after refresh, and model request and tool-call round limits are configurable.

### Deployment

- Deployments support pull policy, stop, rebuild, and other options, and operation history survives resource deletion.

### CI

- A read-only CI query MCP service exposes pipelines, runs, and artifacts.

## 2026-08-05

### Pipelines

- Pipeline execution limits apply from unified configuration, and tasks that exceed them fail with clear reasons.
- Build artifacts can create new application versions.

## 2026-08-04

### Repositories

- Repositories support local directories, enabling builds without a remote Git service.

### Interface

- Loading and empty states use shared components, and environment variables can be edited in place.

## 2026-08-03

### RAGFlow

- RAGFlow split and integrated deployments share conventions, and services are reached through the gateway.

## 2026-08-02

### Runtime

- Service runtime configuration is layered into version and service, and services can override version defaults.
- Controlled-file mounts write correct content and permissions during deployment.

### Logs

- HTTP access logging has its own switch and records query parameters.

## 2026-07-31

### RAGFlow

- RAGFlow includes built-in TEI CPU/GPU variants that deploy with application versions.

### Forms

- Business form validation gives consistent feedback and points errors to specific fields.

## 2026-07-30

### Routes

- Routes belong to services and support service-level multi-domain HTTP/TCP entry points.

### Versions

- Versions export their topology, and component creation is simplified.

## 2026-07-29

### Application versions

- Versions manage runtime configuration, with component and environment-variable editing centralized in version details.

## 2026-07-28

### Application versions

- Versions contain structured component configuration, and service runtime configuration is editable in place on the details page.
- Deployed containers are grouped by version, making the mapping between versions and running instances clear.

### RAGFlow

- RAGFlow supports split deployment and a built-in TEI CPU variant.

## 2026-07-27

### Services

- Services aggregate runtime configuration, and deployment details show structured container status.
- Versions support Compose previews, and services edit ports, environment variables, mounts, and mount content.

### MCP

- Direct deployment tools read stopped Compose containers, confirming service state after a stop.

## 2026-07-26

### Application version model

- Applications gain editable, publishable versions containing components and the Compose topology; services bind a specific version and keep runtime overrides.
- Versions support preview, copy, lifecycle management, and deployment, and multiple services in a project can bind different versions.
- Project environment and ingress policy settings are centralized in project settings, and version deployment generates Compose for the target environment.

### Gateway

- A gateway application type is added: the platform creates and manages the Traefik gateway's version and default service, which can be deployed, stopped, and viewed.

### Runtime

- A local MCP deployment control service generates installation documentation and management scripts locally and handles image pulls, upgrades, and backups.
- RAGFlow split-deployment assets let Orbit manage multi-component deployments.
- Component runtime settings are managed uniformly by versions.

### API

- HTTP errors follow a unified contract, and the frontend shows the error reason directly.

## 2026-07-25

### Versions

- Component configuration dialogs focus on one setting, reducing full-page editing.

## 2026-07-24

### CD refactor

- CD and CI split frontend and backend modules by domain, storage moves from custom repositories to SQLC, and migrations are rebuilt per domain.
- Protocols and DTOs are organized by domain, and frontend and backend use generated types.
- Settings gain scope and navigation coverage, and lists and details unify toolbars, pagination, loading, and error feedback.
- The standalone environments collapse into project execution environments, with project workspaces managed centrally.

## 2026-07-23

### Services

- Service list and detail runtime consoles show container status, copy deployment commands, and read logs.
- Versions split into list and detail views, application pages stay concise, and services configure routes independently.

### Gateway

- Gateway creation, editing, and deploy/stop become separate pages, with ingress policy configured on the gateway.

### Deployment

- Deployment failures write their reasons to the error message and detail logs.

## 2026-07-22

### Versions and deployment

- Versions edit ports, environment variables, mounts, and mount content, and deployment generates Compose from the version configuration.
- Services bind different versions, and one application can have several services.

### Gateway

- Gateways configure domains and pull policy, and exposed applications join the gateway network.
- Routes are managed through the Traefik REST provider.

## 2026-07-21

### Deployment

- Deployment failures show the full reason in the error message and detail logs.

## 2026-07-14

### API

- Routes register by resource, and generated code and SQLC verification are part of unified checks.

### Interface

- CI/CD page layouts are stable at 1440px width.

## 2026-07-13

### Backend

- Startup wiring moves into bootstrap, clarifying application lifecycle responsibilities.

## 2026-07-12

### Backend

- CI/CD execution moves into the application layer, leaving HTTP handlers responsible only for request conversion.

## 2026-07-11

### Backend

- HTTP mapping and SQLC verification boundaries are unified, and invalid requests are rejected at the adapter layer.

## 2026-07-10

### Deployment

- Deployment detail refresh uses one lifecycle and still fetches the latest state after page switches.

## 2026-07-09

### Backend

- Domain subpackages split into usecase, dto, port, and runner, decoupling execution logs, queues, and workspaces from infrastructure.

## 2026-07-08

### API and storage

- The HTTP transport moves to Gin, protocol definitions move to Proto and are organized by domain, and frontend and backend use generated DTOs.
- Migrations move to golang-migrate, and SQLC generates repository queries; request and response contracts are unified per resource.
- HTTP access logging can filter successful static-asset requests.

## 2026-07-07

### API

- Protocol definitions, module paths, and generated code locations are unified under v1.
- CI variable rules move to a shared package, and workspace path handling is unified.

## 2026-07-06

### Database

- SQLC generates repository queries, and migrations move to golang-migrate.

## 2026-07-05

### Database

- Migrations move from a custom runner to golang-migrate, applying upgrades and rollbacks in version order.

## 2026-07-03

### Runtime

- The server configuration provides the runtime API address, and the frontend supports a configurable public address and API prefix.

## 2026-07-02

### Runtime

- Server configuration provides the web access address and CORS cache settings, so the frontend no longer configures the API origin separately.

## 2026-07-01

### API

- HTTP request and response contracts are standardized, and the frontend supports the runtime API address and CORS.

### CD

- Multi-domain Traefik route labels are supported, and deployment logs show container output.

## 2026-06-30

### CD

- Deployment details show the deployment command and container logs, and deployments can force-recreate.

## 2026-06-29

### Configuration

- Configuration loading has a single entry point, and secrets are strictly validated at startup.

## 2026-06-28

### Settings

- System settings become plain form fields, and the webhook signing key is visible.

## 2026-06-27

### CD

- An application creation wizard fills components and deployment configuration from a template.

## 2026-06-26

### CD

- Application management APIs align with resource contracts, and creation and editing share the same fields.

## 2026-06-25

### Logs

- Execution log reads and writes go through LogStore, and stage logs are saved incrementally per file.

## 2026-06-24

### Images and deployment

- Docker images use the Go backend and support separate API and Worker roles.
- CI checkout injects Git credentials, and repository details show built-in variable declarations.

## 2026-06-23

### Contribution

- Code style uses space indentation with an editor configuration.

## 2026-06-22

### Deployment

- Compose bind mounts use physical host paths instead of container-internal paths.

## 2026-06-18

### CI templates

- Templates move to Liquid, and runtime variables and script execution follow the new syntax.

### CI snapshots

- Triggers create missing pipeline snapshots automatically, decoupling executed definitions from later edits.

## 2026-06-17

### Backend

- CD applications and deployments follow a layered architecture, with repository queries separate from handlers.

## 2026-06-16

### Backend

- Project, settings, CI, and authentication modules follow DDD layers, while the frontend keeps the same API.

## 2026-06-15

### Routes

- Traefik routes are managed through platform APIs, and route configuration files are generated when environments start.

### CD

- Application management and deployment APIs create, deploy, and show history.

### CI

- Run, snapshot, and artifact query APIs are added, and credentials are encrypted with Fernet.

## 2026-06-14

### Authentication

- Login uses Cloudflare Turnstile instead of a custom captcha, and security configuration is validated at startup.

### Logs

- HTTP access logging and rotating file logs are configurable.

## 2026-06-13

### Backend

- Backend tests are reorganized by domain, covering CI, CD, and authentication boundaries.

## 2026-06-09

### Backend

- A MySQL-backed Go backend replaces the Python implementation while using the same database and API contracts.

## 2026-06-08

### Login and permissions

- Users, roles, and RBAC permission management are added, with pages and APIs controlled by permissions.

### CD

- Applications support import, export, copying, and template-based creation, and deployment status and logs are tracked in details.

## 2026-05-20

### Users

- User details show role permissions and allow editing usernames and selected fields.
- Route access checks project membership, so non-members cannot view or change project entry points.

## 2026-05-19

### Permissions

- Project membership determines CI and CD resource visibility, including list APIs.

## 2026-05-18

### Projects and permissions

- Project details and member management are added, with project membership as the CI/CD data isolation boundary.

## 2026-05-17

### Repository

- The local sync script no longer uploads personal assistant configuration, reducing accidental inclusion of local files.

## 2026-05-16

### Projects

- Projects support soft deletion: removed from selectors while history stays traceable.
- `project_id` is passed explicitly through every layer, so project isolation no longer relies on implicit context.

## 2026-05-15

### Backend

- CI query, update, and delete APIs carry the project explicitly instead of inferring it from the path.

## 2026-05-14

### Projects

- Project management and multi-tenant isolation are added, with CI and CD resources owned by projects.

## 2026-05-13

### Login security

- Login adds CSRF protection, rate limiting, and account status checks, and every login attempt requires a captcha.
- Google OAuth login is added with generic provider configuration.

## 2026-05-12

### Frontend

- The interface supports Chinese and English plus light and dark themes, and pages adapt to mobile and tablet.
- Monaco Editor shows pipeline stage logs and edits configuration.
- CD application configuration becomes structured JSON, and CI stage definitions follow suit.

## 2026-05-11

### Frontend

- Static assets and SPA routing coexist, so refreshing a detail page still works.

## 2026-05-10

### Frontend

- A shared design system and theme switching are introduced, with dashboards using data cards and consistent empty states.
- Dialogs, pagination, and code editors use consistent interactions, and CI template and run pages are localized.

## 2026-05-09

### Deployment

- The deployment list filters by application, and deployment records keep complete state and logs.

### Interface

- Lists, details, and dialogs use consistent toolbars, buttons, and pagination.

## 2026-05-08

### Interface

- The frontend moves to a shared component library with consistent tables, badges, and time display.
- CD API paths use singular resource forms.

## 2026-05-07

### Interface

- The frontend moves to reka-ui, making components and styles consistently reusable.

## 2026-05-06

### Interface

- reka-ui component documentation and the style-processing script are introduced.

## 2026-05-04

### CI

- Build-stage volume mounts become structured data instead of hand-written strings in scripts.

## 2026-05-03

### Repositories

- Updating repository basics keeps custom variables.

## 2026-05-02

### Repositories

- A Gitee credential type is added, and updating a repository keeps existing variables.

## 2026-04-30

### Operations

- Operations scripts support GHCR authentication for pulling private images.

## 2026-04-18

### CD

- A Compose preview API shows the generated result before deployment.

## 2026-04-17

### CD

- Applications support service-level configuration, with multiple services sharing an application version.

## 2026-04-15

### CD

- Application route management configures dynamic Traefik routes from application details.
- A deployment log SSE endpoint streams deployment output in real time.

## 2026-04-14

### CI

- A built-in runtime date variable is added, and the stage log drawer is resizable.

### CD

- Background tasks use a dependency-injection container, so new tasks no longer pollute global state.

## 2026-04-13

### CI

- Pipeline runs, templates, and artifacts support search, filtering, and pagination.
- Artifact declarations appear on template and snapshot details, making build output verifiable.

## 2026-04-12

### CI

- Pipeline runs stream container logs in real time and clean up containers afterwards.
- Git credentials are checked before triggering, and runs stop when they are missing.

## 2026-04-11

### CI

- Variables support defaults, trigger-time overrides, and editable declarations.
- Runs stop when templates have unsaved changes, preventing execution of unconfirmed definitions.

## 2026-04-10

### CI

- Pipelines save templates, orchestration, and variables separately, and runs follow the snapshot.
- Templates and stages can be copied, and deleting a template cleans up its instances.

## 2026-04-09

### CI

- Run details follow the snapshot and show the template version and error message.
- The CI "project" is renamed to "repository" across the interface and API.

## 2026-04-08

### CI

- Templates support a version field, and run details show template and snapshot versions.
- Stage logs are written incrementally per file, and canceled runs are marked correctly.

## 2026-04-07

### CI

- Stages and pipeline snapshots are introduced, so runs pin the definition used for that run.

## 2026-04-03

### CI

- Pipeline templates, stage orchestration, and variables save separately, and snapshots provide stable run-time definitions.
- Checkout stages that need a repository are blocked when credentials are not configured.

## 2026-04-02

### Interface

- The frontend moves to Tailwind and DaisyUI, and CI/CD navigation and pages are split by module.

## 2026-04-01

### CI launch

- Repositories, credentials, stage templates, and pipeline templates are managed, with GitHub and Gitee credentials.
- Pipelines run stages in parallel along a dependency graph, with fail-fast, retry skipping completed stages, run cancellation, and incremental logs.
- Runs save template and variable snapshots, and artifacts are archived per run and passed between pipelines.
- New CI pages show run status, stage logs, and artifacts.

## 2026-03-30

### Documentation

- The documentation directory is rebuilt, with exploratory discussions grouped by topic.

## 2026-03-28

### CD

- Application cards show available actions by state, and deployment logs and shell file line endings display correctly.
- Overridden settings are clearly marked in system settings.

## 2026-03-27

### CD

- Applications and deployments share one state machine, and the frontend follows the new state model.

### Build

- A China-mirror build option is provided, and initialization migrations no longer overwrite existing ACME certificates.

## 2026-03-26

### CD

- Applications support import and export, with a preview that checks name conflicts before import.

### Operations

- Management scripts are unified as `manage.py`, a backup command is added, and route certificate configuration is generated per environment.

## 2026-03-25

### Authentication

- Login uses email authentication, and Google OAuth can be proxied through a Cloudflare Worker.

## 2026-03-24

### Initial release

- The first runnable version provides a web management interface, Docker Compose deployment, dynamic Traefik routing, and application deployment state.
- Container images and a local run method are provided, with an initial admin account and baseline migrations.

### Operations

- Certificate renewal and backup scripts are provided, and updates keep `.env` by default.
