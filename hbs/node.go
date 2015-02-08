package hbs

import (
	"github.com/dinp/common/model"
	"github.com/dinp/server/g"
	"time"
)

type NodeState int

func (this *NodeState) Push(req *model.NodeRequest, resp *model.NodeResponse) error {
	if req == nil {
		resp.Code = 1
		return nil
	}

	g.UpdateNode(&req.Node)

	if req.Containers == nil || len(req.Containers) == 0 {
		return nil
	}

	now := time.Now().Unix()

	for _, dto := range req.Containers {
		container := &model.Container{
			Id:       dto.Id,
			Ip:       req.Ip,
			Image:    dto.Image,
			AppName:  dto.AppName,
			Ports:    dto.Ports,
			Status:   dto.Status,
			UpdateAt: now,
		}
		g.RealState.UpdateContainer(container)
	}

	return nil
}

func (this *NodeState) NodeDown(ip string, resp *model.NodeResponse) error {
	if ip == "" {
		resp.Code = 1
		return nil
	}

	g.DeleteNode(ip)
	g.RealState.DeleteByIp(ip)
	return nil
}
