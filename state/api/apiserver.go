package api

import (
	"code.google.com/p/go.net/websocket"
	"errors"
	"fmt"
	"launchpad.net/juju-core/state"
	"launchpad.net/juju-core/state/statecmd"
	"sync"
)

// TODO(rog) remove this when the rest of the system
// has been updated to set passwords appropriately.
var AuthenticationEnabled = false

var (
	errBadId       = errors.New("id not found")
	errBadCreds    = errors.New("invalid entity name or password")
	errNotLoggedIn = errors.New("not logged in")
	errPerm        = errors.New("permission denied")
)

// srvRoot represents a single client's connection to the state.
type srvRoot struct {
	admin  *srvAdmin
	client *srvClient
	srv    *Server
	conn   *websocket.Conn

	user authUser
}

// srvAdmin is the only object that unlogged-in
// clients can access. It holds any methods
// that are needed to log in.
type srvAdmin struct {
	root *srvRoot
}

// srvMachine serves API methods on a machine.
type srvMachine struct {
	root *srvRoot
	m    *state.Machine
}

// srvUnit serves API methods on a unit.
type srvUnit struct {
	root *srvRoot
	u    *state.Unit
}

// srvUser serves API methods on a state User.
type srvUser struct {
	root *srvRoot
	u    *state.User
}

// srvClient serves client-specific API methods.
type srvClient struct {
	root *srvRoot
}

func newStateServer(srv *Server, conn *websocket.Conn) *srvRoot {
	r := &srvRoot{
		srv:  srv,
		conn: conn,
	}
	r.admin = &srvAdmin{
		root: r,
	}
	r.client = &srvClient{
		root: r,
	}
	return r
}

func (r *srvRoot) Admin(id string) (*srvAdmin, error) {
	if id != "" {
		// Safeguard id for possible future use.
		return nil, errBadId
	}
	return r.admin, nil
}

// requireAgent checks whether the current client is an agent and hence
// may access the agent APIs.  We filter out non-agents when calling one
// of the accessor functions (Machine, Unit, etc) which avoids us making
// the check in every single request method.
func (r *srvRoot) requireAgent() error {
	e := r.user.entity()
	if e == nil {
		return errNotLoggedIn
	}
	if !isAgent(e) {
		return errPerm
	}
	return nil
}

// requireClient returns an error unless the current
// client is a juju client user.
func (r *srvRoot) requireClient() error {
	e := r.user.entity()
	if e == nil {
		return errNotLoggedIn
	}
	if isAgent(e) {
		return errPerm
	}
	return nil
}

func (r *srvRoot) Machine(id string) (*srvMachine, error) {
	if err := r.requireAgent(); err != nil {
		return nil, err
	}
	m, err := r.srv.state.Machine(id)
	if err != nil {
		return nil, err
	}
	return &srvMachine{
		root: r,
		m:    m,
	}, nil
}

func (r *srvRoot) Unit(name string) (*srvUnit, error) {
	if err := r.requireAgent(); err != nil {
		return nil, err
	}
	u, err := r.srv.state.Unit(name)
	if err != nil {
		return nil, err
	}
	return &srvUnit{
		root: r,
		u:    u,
	}, nil
}

func (r *srvRoot) User(name string) (*srvUser, error) {
	// Any user is allowed to access their own user object.
	// We check at this level rather than at the operation
	// level to stop malicious probing for current user names.
	// When we provide support for user administration,
	// this will need to be changed to allow access to
	// the administrator.
	e := r.user.entity()
	if e == nil {
		return nil, errNotLoggedIn
	}
	if e.EntityName() != name {
		return nil, errPerm
	}
	u, err := r.srv.state.User(name)
	if err != nil {
		return nil, err
	}
	return &srvUser{
		root: r,
		u:    u,
	}, nil
}

func (r *srvRoot) Client(id string) (*srvClient, error) {
	if err := r.requireClient(); err != nil {
		return nil, err
	}
	if id != "" {
		// Safeguard id for possible future use.
		return nil, errBadId
	}
	return r.client, nil
}

func (c *srvClient) Status() (Status, error) {
	ms, err := c.root.srv.state.AllMachines()
	if err != nil {
		return Status{}, err
	}
	status := Status{
		Machines: make(map[string]MachineInfo),
	}
	for _, m := range ms {
		instId, _ := m.InstanceId()
		status.Machines[m.Id()] = MachineInfo{
			InstanceId: string(instId),
		}
	}
	return status, nil
}

func (c *srvClient) SetConfig(p statecmd.SetConfigParams) error {
	return statecmd.SetConfig(c.root.srv.state, p)
}

type rpcCreds struct {
	EntityName string
	Password   string
}

// Login logs in with the provided credentials.
// All subsequent requests on the connection will
// act as the authenticated user.
func (a *srvAdmin) Login(c rpcCreds) error {
	return a.root.user.login(a.root.srv.state, c.EntityName, c.Password)
}

type rpcMachine struct {
	InstanceId string
}

// Get retrieves all the details of a machine.
func (m *srvMachine) Get() (info rpcMachine) {
	instId, _ := m.m.InstanceId()
	info.InstanceId = string(instId)
	return
}

type rpcPassword struct {
	Password string
}

func setPassword(e state.AuthEntity, password string) error {
	// Catch expected common case of mispelled
	// or missing Password parameter.
	if password == "" {
		return fmt.Errorf("password is empty")
	}
	return e.SetPassword(password)
}

// SetPassword sets the machine's password.
func (m *srvMachine) SetPassword(p rpcPassword) error {
	// Allow:
	// - the machine itself.
	// - the environment manager.
	e := m.root.user.entity()
	allow := e.EntityName() == m.m.EntityName() ||
		isMachineWithJob(e, state.JobManageEnviron)
	if !allow {
		return errPerm
	}
	return setPassword(m.m, p.Password)
}

// Get retrieves all the details of a unit.
func (u *srvUnit) Get() (rpcUnit, error) {
	var ru rpcUnit
	ru.DeployerName, _ = u.u.DeployerName()
	// TODO add other unit attributes
	return ru, nil
}

// SetPassword sets the unit's password.
func (u *srvUnit) SetPassword(p rpcPassword) error {
	ename := u.root.user.entity().EntityName()
	// Allow:
	// - the unit itself.
	// - the machine responsible for unit, if unit is principal
	// - the unit's principal unit, if unit is subordinate
	allow := ename == u.u.EntityName()
	if !allow {
		deployerName, ok := u.u.DeployerName()
		allow = ok && ename == deployerName
	}
	if !allow {
		return errPerm
	}
	return setPassword(u.u, p.Password)
}

type rpcUnit struct {
	DeployerName string
	// TODO(rog) other unit attributes.
}

// SetPassword sets the user's password.
func (u *srvUser) SetPassword(p rpcPassword) error {
	return setPassword(u.u, p.Password)
}

type rpcUser struct {
	// This is a placeholder for any information
	// that may be associated with a user in the
	// future.
}

// Get retrieves all details of a user.
func (u *srvUser) Get() (rpcUser, error) {
	return rpcUser{}, nil
}

// authUser holds login details. It's ok to call
// its methods concurrently.
type authUser struct {
	mu      sync.Mutex
	_entity state.AuthEntity // logged-in entity (access only when mu is locked)
}

// login authenticates as entity with the given name,.
func (u *authUser) login(st *state.State, entityName, password string) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	entity, err := st.AuthEntity(entityName)
	if err != nil && !state.IsNotFound(err) {
		return err
	}
	// TODO(rog) remove
	if !AuthenticationEnabled {
		u._entity = entity
		return nil
	}
	// We return the same error when an entity
	// does not exist as for a bad password, so that
	// we don't allow unauthenticated users to find information
	// about existing entities.
	if err != nil || !entity.PasswordValid(password) {
		return errBadCreds
	}
	u._entity = entity
	return nil
}

// entity returns the currently logged-in entity, or nil if not
// currently logged on.  The returned entity should not be modified
// because it may be used concurrently.
func (u *authUser) entity() state.AuthEntity {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u._entity
}

// isMachineWithJob returns whether the given entity is a machine that
// is configured to run the given job.
func isMachineWithJob(e state.AuthEntity, j state.MachineJob) bool {
	m, ok := e.(*state.Machine)
	if !ok {
		return false
	}
	for _, mj := range m.Jobs() {
		if mj == j {
			return true
		}
	}
	return false
}

// isAgent returns whether the given entity is an agent.
func isAgent(e state.AuthEntity) bool {
	_, isUser := e.(*state.User)
	return !isUser
}
