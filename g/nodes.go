package g

import (
	"github.com/dinp/common/model"
	"log"
	"sort"
	"sync"
	"time"
)

var (
	Nodes     = make(map[string]*model.Node)
	NodeMutex = new(sync.RWMutex)
)

func Clone() map[string]*model.Node {
	ret := make(map[string]*model.Node)
	NodeMutex.RLock()
	defer NodeMutex.RUnlock()
	for ip, n := range Nodes {
		ret[ip] = n
	}
	return ret
}

func UpdateNode(node *model.Node) {
	node.UpdateAt = time.Now().Unix()
	NodeMutex.Lock()
	defer NodeMutex.Unlock()
	Nodes[node.Ip] = node
}

func DeleteStaleNode(before int64) {
	need_delete := make([]*model.Node, 0)
	NodeMutex.RLock()
	for _, node := range Nodes {
		if node.UpdateAt < before {
			need_delete = append(need_delete, node)
		}
	}
	NodeMutex.RUnlock()

	NodeMutex.Lock()
	for _, node := range need_delete {
		log.Printf("[NodeDown] ip: %s", node.Ip)
		delete(Nodes, node.Ip)
	}
	NodeMutex.Unlock()
}

func DeleteNode(ip string) {
	NodeMutex.Lock()
	defer NodeMutex.Unlock()
	delete(Nodes, ip)
}

func NodeCount() int {
	NodeMutex.RLock()
	defer NodeMutex.RUnlock()
	return len(Nodes)
}

func TheOne() *model.Node {
	NodeMutex.RLock()
	defer NodeMutex.RUnlock()
	for _, n := range Nodes {
		return n
	}
	return nil
}

func GetNode(ip string) *model.Node {
	NodeMutex.RLock()
	defer NodeMutex.RUnlock()
	return Nodes[ip]
}

func ChooseNode(app *model.App, deployCnt int) map[string]int {
	ret := make(map[string]int)
	size := NodeCount()
	if size == 0 {
		return ret
	}

	if size == 1 {
		n := TheOne()
		if n != nil && n.MemFree > uint64(deployCnt*app.Memory) {
			ret[n.Ip] = deployCnt
			return ret
		}
		log.Printf(">>> memory not enough for %d instance of %s <<<", app.InstanceCnt, app.Name)
		return ret
	}

	copyNodes := Clone()

	// order by MemFree desc
	ns := make(model.NodeSlice, 0, size)
	for _, n := range copyNodes {
		ns = append(ns, n)
	}

	sort.Sort(ns)

	// delete node which MemFree < app.Memory
	memFreeIsOK := make([]*model.Node, 0, size)
	for _, n := range ns {
		if n.MemFree > uint64(app.Memory) {
			memFreeIsOK = append(memFreeIsOK, n)
		}
	}

	size = len(memFreeIsOK)
	if size == 0 {
		return ret
	}

	// node not enough
	if size < deployCnt {

		// every node at least create one container
		for _, n := range memFreeIsOK {
			ret[n.Ip] = 1
		}

		done := len(memFreeIsOK)
		for {
			for _, n := range memFreeIsOK {
				ret[n.Ip] += 1
				done++
				if done == deployCnt {
					goto CHK_MEM
				}
			}
		}

	CHK_MEM:

		for _, n := range memFreeIsOK {
			if n.MemFree < uint64(app.Memory*ret[n.Ip]) {
				log.Printf(">>> memory not enough for %d instance of %s <<<", app.InstanceCnt, app.Name)
				return make(map[string]int)
			}
		}

		return ret
	}

	if size == deployCnt {
		for _, n := range memFreeIsOK {
			ret[n.Ip] = 1
		}
		return ret
	}

	// node enough
	has_deployed_count := app.InstanceCnt - deployCnt
	if has_deployed_count == 0 {
		// first deploy
		done := 0
		for _, n := range memFreeIsOK {
			ret[n.Ip] = 1
			done++
			if done == deployCnt {
				return ret
			}
		}
	}

	// we have enough nodes. delete the node which has deployed this app.
	// we can delete a maximum of size - deployCnt
	can_delete_node_count := size - deployCnt
	// the nodes not deploy this app, order by MemFree asc
	not_deploy_this_app := make([]*model.Node, 0)

	all_ok := false
	for i := size - 1; i >= 0; i-- {
		if all_ok {
			not_deploy_this_app = append(not_deploy_this_app, memFreeIsOK[i])
			continue
		}

		if !RealState.HasRelation(app.Name, memFreeIsOK[i].Ip) {
			not_deploy_this_app = append(not_deploy_this_app, memFreeIsOK[i])
			continue
		}

		if can_delete_node_count > 0 {
			has_deployed_count--
			if has_deployed_count == 0 {
				// the rest nodes are all not deploy this app
				all_ok = true
				continue
			}
			can_delete_node_count--
		} else {
			not_deploy_this_app = append(not_deploy_this_app, memFreeIsOK[i])
			all_ok = true
		}
	}

	// order by MemFree desc
	cnt := 0
	for i := len(not_deploy_this_app) - 1; i >= 0; i-- {
		ret[not_deploy_this_app[i].Ip] = 1
		cnt++
		if cnt == deployCnt {
			return ret
		}
	}

	return make(map[string]int)
}
