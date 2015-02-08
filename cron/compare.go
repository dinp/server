package cron

import (
	"fmt"
	"github.com/dinp/common/model"
	"github.com/dinp/server/g"
	"github.com/fsouza/go-dockerclient"
	"github.com/toolkits/slice"
	"log"
	"strings"
	"time"
)

func getDesiredState() (map[string]*model.App, error) {
	sql := "select name, memory, instance, image, status from app where status = 0 and image <> ''"
	rows, err := g.DB.Query(sql)
	if err != nil {
		log.Printf("[ERROR] exec %s fail: %s", sql, err)
		return nil, err
	}

	var desiredState = make(map[string]*model.App)
	for rows.Next() {
		var app model.App
		err = rows.Scan(&app.Name, &app.Memory, &app.InstanceCnt, &app.Image, &app.Status)
		if err != nil {
			log.Printf("[ERROR] %s scan fail: %s", sql, err)
			return nil, err
		}

		desiredState[app.Name] = &app
	}

	return desiredState, nil
}

func CompareState() {
	duration := time.Duration(g.Config().Interval) * time.Second
	time.Sleep(duration)
	for {
		time.Sleep(duration)
		compareState()
	}
}

func compareState() {
	desiredState, err := getDesiredState()
	if err != nil {
		log.Println("[ERROR] get desired state fail:", err)
		return
	}

	debug := g.Config().Debug

	if debug {
		log.Println("comparing......")
	}

	if len(desiredState) == 0 {
		if debug {
			log.Println("no desired app. do nothing")
		}
		// do nothing.
		return
	}

	newAppSlice := []string{}

	for name, app := range desiredState {
		if !g.RealState.RealAppExists(name) {
			if debug && app.InstanceCnt > 0 {
				log.Println("[=-NEW-=]:", name)
			}
			newAppSlice = append(newAppSlice, name)
			createNewContainer(app, app.InstanceCnt)
		}
	}

	realNames := g.RealState.Keys()

	for ii, name := range realNames {
		if debug {
			log.Printf("#%d: %s", ii, name)
		}

		if slice.ContainsString(newAppSlice, name) {
			continue
		}

		app, exists := desiredState[name]
		if !exists {
			if debug {
				log.Println("[=-DEL-=]:", name)
			}
			dropApp(name)
			continue
		}

		sa, _ := g.RealState.GetSafeApp(name)
		isOld, olds := sa.IsOldVersion(app.Image)
		if isOld {
			if len(olds) > 0 || app.InstanceCnt > 0 {
				log.Println("[=-UPGRADE-=]")
			}
			// deploy new instances
			createNewContainer(app, app.InstanceCnt)
			// delete old instances
			for _, c := range olds {
				dropContainer(c)
			}

			continue
		}

		nowCnt := sa.ContainerCount()

		if nowCnt < app.InstanceCnt {
			if debug {
				log.Printf("add:%d", app.InstanceCnt-nowCnt)
			}
			createNewContainer(app, app.InstanceCnt-nowCnt)
			continue
		}

		if nowCnt > app.InstanceCnt {
			if debug {
				log.Printf("del:%d", nowCnt-app.InstanceCnt)
			}
			dropContainers(sa.Containers(), nowCnt-app.InstanceCnt)
		}
	}
}

func createNewContainer(app *model.App, deployCnt int) {
	if deployCnt == 0 {
		return
	}

	if app.Status != model.AppStatus_Success {
		if g.Config().Debug {
			log.Printf("!!! App=%s Status = %d", app.Name, app.Status)
		}
		return
	}

	ip_count := g.ChooseNode(app, deployCnt)
	if len(ip_count) == 0 {
		log.Println("no node..zZ")
		return
	}

	for ip, count := range ip_count {
		for k := 0; k < count; k++ {
			DockerRun(app, ip)
		}
	}
}

func dropApp(appName string) {
	if appName == "" {
		return
	}

	if g.Config().Debug {
		log.Println("drop app:", appName)
	}

	sa, _ := g.RealState.GetSafeApp(appName)
	cs := sa.Containers()
	for _, c := range cs {
		dropContainer(c)
	}
	g.RealState.DeleteSafeApp(appName)

	rc := g.RedisConnPool.Get()
	defer rc.Close()

	uriKey := fmt.Sprintf("%s%s.%s", g.Config().Redis.RsPrefix, appName, g.Config().Domain)
	rc.Do("DEL", uriKey)
}

func dropContainers(cs []*model.Container, cnt int) {
	if cnt == 0 {
		return
	}

	done := 0
	for _, c := range cs {
		dropContainer(c)
		done++
		if done == cnt {
			break
		}
	}
}

func dropContainer(c *model.Container) {

	if g.Config().Debug {
		log.Println("drop container:", c)
	}

	addr := fmt.Sprintf("http://%s:%d", c.Ip, g.Config().DockerPort)
	client, err := docker.NewClient(addr)
	if err != nil {
		log.Println("docker.NewClient fail:", err)
		return
	}

	err = client.RemoveContainer(docker.RemoveContainerOptions{ID: c.Id, Force: true})
	if err != nil {
		log.Println("docker.RemoveContainer fail:", err)
		return
	}

	// remember to delete real state map item
	sa, exists := g.RealState.GetSafeApp(c.AppName)
	if exists {
		sa.DeleteContainer(c)
	}
}

func BuildEnvArray(envVars map[string]string) []string {
	size := len(envVars)
	if size == 0 {
		return []string{}
	}

	arr := make([]string, size)
	idx := 0
	for k, v := range envVars {
		arr[idx] = fmt.Sprintf("%s=%s", k, v)
		idx++
	}

	return arr
}

func ParseRepositoryTag(repos string) (string, string) {
	n := strings.LastIndex(repos, ":")
	if n < 0 {
		return repos, ""
	}
	if tag := repos[n+1:]; !strings.Contains(tag, "/") {
		return repos[:n], tag
	}
	return repos, ""
}

func DockerRun(app *model.App, ip string) {
	if g.Config().Debug {
		log.Printf("create container. app:%s, ip:%s\n", app.Name, ip)
	}

	envVars, err := g.LoadEnvVarsOf(app.Name)
	if err != nil {
		log.Println("[ERROR] load env fail:", err)
		return
	}

	envVars["APP_NAME"] = app.Name
	envVars["HOST_IP"] = ip
	if g.Config().Scribe.Ip != "" {
		envVars["SCRIBE_IP"] = g.Config().Scribe.Ip
	} else {
		envVars["SCRIBE_IP"] = ip
	}
	envVars["SCRIBE_PORT"] = fmt.Sprintf("%d", g.Config().Scribe.Port)

	addr := fmt.Sprintf("http://%s:%d", ip, g.Config().DockerPort)

	client, err := docker.NewClient(addr)
	if err != nil {
		log.Println("[ERROR] docker.NewClient fail:", err)
		return
	}

	opts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Memory: int64(app.Memory * 1024 * 1024),
			ExposedPorts: map[docker.Port]struct{}{
				docker.Port("8080/tcp"): {},
			},
			Image:        app.Image,
			AttachStdin:  false,
			AttachStdout: false,
			AttachStderr: false,
			Env:          BuildEnvArray(envVars),
		},
	}

	container, err := client.CreateContainer(opts)

	if err != nil {
		if err == docker.ErrNoSuchImage {
			repos, tag := ParseRepositoryTag(app.Image)
			e := client.PullImage(docker.PullImageOptions{Repository: repos, Tag: tag}, docker.AuthConfiguration{})
			if e != nil {
				log.Println("[ERROR] pull image", app.Image, "fail:", e)
				return
			}

			// retry
			container, err = client.CreateContainer(opts)
			if err != nil {
				log.Println("[ERROR] retry create container fail:", err, "ip:", ip)
				g.UpdateAppStatus(app, model.AppStatus_CreateContainerFail)
				return
			}
		} else {
			log.Println("[ERROR] create container fail:", err, "ip:", ip)
			if err != nil && strings.Contains(err.Error(), "cannot connect") {
				g.DeleteNode(ip)
				g.RealState.DeleteByIp(ip)
				return
			}
			g.UpdateAppStatus(app, model.AppStatus_CreateContainerFail)
			return
		}
	}

	err = client.StartContainer(container.ID, &docker.HostConfig{
		PortBindings: map[docker.Port][]docker.PortBinding{
			"8080/tcp": []docker.PortBinding{docker.PortBinding{}},
		},
	})

	if err != nil {
		log.Println("[ERROR] docker.StartContainer fail:", err)
		g.UpdateAppStatus(app, model.AppStatus_StartContainerFail)
		return
	}

	if g.Config().Debug {
		log.Println("start container success:-)")
	}

}
