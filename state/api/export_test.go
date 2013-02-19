package api
import "launchpad.net/juju-core/rpc"

// RPCClient returns the RPC client for the state, so that testing
// functions can tickle parts of the API that the conventional entry
// points don't reach.
func (st *State) RPCClient() *rpc.Client {
	return st.client
}
