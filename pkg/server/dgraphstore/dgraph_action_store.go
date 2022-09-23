package dgraphstore

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"golang.org/x/net/context"
)

type DgraphActionStore struct {
	dgraphClient *dgo.Dgraph
}

func NewDgraphActionStore(dgraphClient *dgo.Dgraph) *DgraphActionStore {
	d := &DgraphActionStore{dgraphClient: dgraphClient}

	return d
}

func (d *DgraphActionStore) GetAll() ([]*tarianpb.Action, error) {
	q := fmt.Sprintf(`
		{
			actions(func: type(Action)) {
				%s
			}
		}
	`, actionFields)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	tx := d.dgraphClient.NewReadOnlyTxn()
	resp, err := tx.Query(ctx, q)
	if err != nil {
		return nil, err
	}

	var respActionList dgraphActionList
	if err := json.Unmarshal(resp.GetJson(), &respActionList); err != nil {
		return nil, err
	}

	actions := respActionList.toPbActions()

	return actions, nil
}

const actionFields = `
	uid
	dgraph.type
	action_namespace
	action_name
	action_selector
	action_on_violated_process
	action_on_violated_file
	action_on_falco_alert
	action_falco_alert_priority
	action_action
`

type dgraphActionList struct {
	Actions []Action
}

func (da *dgraphActionList) toPbActions() []*tarianpb.Action {
	pbActions := []*tarianpb.Action{}

	for _, dgraphAction := range da.Actions {
		pbAction := tarianpb.NewAction()
		pbAction.Namespace = dgraphAction.Namespace
		pbAction.Name = dgraphAction.Name
		json.Unmarshal([]byte(dgraphAction.Selector), &pbAction.Selector)
		pbAction.OnViolatedProcess = dgraphAction.OnViolatedProcess
		pbAction.OnViolatedFile = dgraphAction.OnViolatedFile
		pbAction.OnFalcoAlert = dgraphAction.OnFalcoAlert

		pbAction.FalcoPriority = tarianpb.FalcoPriority(dgraphAction.FalcoAlertPriority)
		pbAction.Action = dgraphAction.Action

		pbActions = append(pbActions, pbAction)
	}

	return pbActions
}

func (d *DgraphActionStore) FindByNamespace(namespace string) ([]*tarianpb.Action, error) {
	q := fmt.Sprintf(`
		query actionQuery($namespace: string) {
			actions(func: type(Action)) @filter(eq(action_namespace, $namespace)) {
				%s
			}
		}
	`, actionFields)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	tx := d.dgraphClient.NewReadOnlyTxn()
	vars := map[string]string{"$namespace": namespace}

	resp, err := tx.QueryWithVars(ctx, q, vars)
	if err != nil {
		return nil, err
	}

	var result dgraphActionList
	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return nil, err
	}

	actions := result.toPbActions()

	return actions, nil
}

func (d *DgraphActionStore) NamespaceAndNameExist(namespace, name string) (bool, error) {
	uid, err := d.findActionUIDByNamespaceAndName(namespace, name)

	if err != nil {
		return false, err
	}

	exist := uid != ""
	return exist, err
}

func (d *DgraphActionStore) Add(action *tarianpb.Action) error {
	dgraphAction, err := dgraphActionFromPb(action)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(dgraphAction)
	if err != nil {
		return err
	}

	mu := &api.Mutation{
		CommitNow: true,
	}

	mu.SetJson = payload
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_, err = d.dgraphClient.NewTxn().Mutate(ctx, mu)
	if err != nil {
		return err
	}

	return nil
}

func dgraphActionFromPb(pbAction *tarianpb.Action) (*Action, error) {
	selectorJSON, err := json.Marshal(pbAction.GetSelector())
	if err != nil {
		return nil, err
	}

	dgraphAction := Action{
		UID:                "_:action",
		DType:              []string{"Action"},
		Namespace:          pbAction.GetNamespace(),
		Name:               pbAction.GetName(),
		Selector:           string(selectorJSON),
		OnViolatedProcess:  pbAction.GetOnViolatedProcess(),
		OnViolatedFile:     pbAction.GetOnViolatedFile(),
		OnFalcoAlert:       pbAction.GetOnFalcoAlert(),
		FalcoAlertPriority: int(pbAction.GetFalcoPriority()),
		Action:             pbAction.GetAction(),
	}

	return &dgraphAction, nil
}

func (d *DgraphActionStore) RemoveByNamespaceAndName(namespace, name string) error {
	uid, err := d.findActionUIDByNamespaceAndName(namespace, name)

	if err != nil {
		return err
	}

	if uid == "" {
		return nil
	}

	q := fmt.Sprintf(`{"uid": "%s"}`, uid)

	op := &api.Mutation{DeleteJson: []byte(q), CommitNow: true}
	txn := d.dgraphClient.NewTxn()

	ctxDiscard, cancelDiscard := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelDiscard()
	defer txn.Discard(ctxDiscard)

	ctxMutate, cancelMutate := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelMutate()
	_, err = txn.Mutate(ctxMutate, op)

	return err
}

func (d *DgraphActionStore) findActionUIDByNamespaceAndName(namespace, name string) (string, error) {
	const q = `
		query actionQuery($namespace: string, $name: string) {
			actions(func: type(Action), first: 1) @filter(eq(action_namespace, $namespace) AND eq(action_name, $name)) {
				uid
			}
		}
	`

	tx := d.dgraphClient.NewReadOnlyTxn()
	vars := map[string]string{"$namespace": namespace, "$name": name}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := tx.QueryWithVars(ctx, q, vars)
	if err != nil {
		return "", err
	}

	var result dgraphActionList
	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return "", err
	}

	if len(result.Actions) == 0 {
		return "", nil
	}

	return result.Actions[0].UID, nil
}
