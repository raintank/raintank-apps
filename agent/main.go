package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
	"github.com/op/go-logging"
	"github.com/raintank/raintank-apps/pkg/message"
	"github.com/raintank/raintank-apps/pkg/session"
	"github.com/raintank/raintank-apps/server/model"
	"github.com/rakyll/globalconf"
)

const Version int = 1

var log = logging.MustGetLogger("default")

var (
	showVersion = flag.Bool("version", false, "print version string")
	logLevel    = flag.Int("log-level", 4, "log level. 5=DEBUG|4=INFO|3=NOTICE|2=WARNING|1=ERROR|0=CRITICAL")
	confFile    = flag.String("config", "/etc/raintank/collector.ini", "configuration file path")

	addr       = flag.String("addr", "localhost:8081", "http service address")
	snapUrlStr = flag.String("snap-url", "http://localhost:8181", "url of SNAP server.")
	nodeName   = flag.String("name", "", "agent name")
)

type TaskCache struct {
	sync.RWMutex
	Tasks     map[int64]*model.TaskDTO
	SnapTasks map[string]*rbody.ScheduledTask
}

func (t *TaskCache) AddTask(task *model.TaskDTO) error {
	t.Lock()
	defer t.Unlock()
	snapTaskName := fmt.Sprintf("raintank-apps:%d", task.Id)
	snapTask, ok := t.SnapTasks[snapTaskName]
	if !ok {
		log.Debugf("New task recieved %s", snapTaskName)
		snapTask, err := CreateSnapTask(task, snapTaskName)
		if err != nil {
			return err
		}
		t.SnapTasks[snapTaskName] = snapTask
	} else {
		log.Debugf("task %s already in the cache.", snapTaskName)
		if task.Updated.After(time.Unix(snapTask.CreationTimestamp, 0)) {
			log.Debugf("%s needs to be updated", snapTaskName)
			// need to update task.
			if err := RemoveSnapTask(snapTask); err != nil {
				return err
			}
			snapTask, err := CreateSnapTask(task, snapTaskName)
			if err != nil {
				return err
			}
			t.SnapTasks[snapTaskName] = snapTask
		}
	}
	t.Tasks[task.Id] = task
	return nil
}

func (t *TaskCache) RemoveTask(task *model.TaskDTO) error {
	t.Lock()
	defer t.Unlock()
	snapTaskName := fmt.Sprintf("raintank-apps:%d", task.Id)
	snapTask, ok := t.SnapTasks[snapTaskName]
	if !ok {
		log.Debugf("task to remove not in cache. %s", snapTaskName)
	} else {
		if err := RemoveSnapTask(snapTask); err != nil {
			return err
		}
		delete(t.SnapTasks, snapTaskName)
	}
	delete(t.Tasks, task.Id)
	return nil
}

func (t *TaskCache) IndexSnapTasks(tasks []*rbody.ScheduledTask) error {
	t.Lock()
	t.SnapTasks = make(map[string]*rbody.ScheduledTask)
	for _, task := range tasks {
		t.SnapTasks[task.Name] = task
	}
	t.Unlock()
	return nil
}

var GlobalTaskCache *TaskCache

func connect(u url.URL) (*websocket.Conn, error) {
	log.Infof("connecting to %s", u.String())
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	return conn, err
}

func main() {
	flag.Parse()
	// Only try and parse the conf file if it exists
	if _, err := os.Stat(*confFile); err == nil {
		conf, err := globalconf.NewWithOptions(&globalconf.Options{Filename: *confFile})
		if err != nil {
			panic(fmt.Sprintf("error with configuration file: %s", err))
		}
		conf.ParseAll()
	}

	logging.SetFormatter(logging.GlogFormatter)
	logging.SetLevel(logging.Level(*logLevel), "default")
	log.SetBackend(logging.AddModuleLevel(logging.NewLogBackend(os.Stdout, "", 0)))

	if *nodeName == "" {
		log.Fatalf("name must be set.")
	}

	snapUrl, err := url.Parse(*snapUrlStr)
	if err != nil {
		log.Fatalf("could not parse snapUrl. %s", err)
	}
	InitSnapClient(snapUrl)
	catalog, err := GetSnapMetrics()
	if err != nil {
		log.Fatal(err)
	}
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	shutdownStart := make(chan struct{})

	GlobalTaskCache = &TaskCache{
		Tasks:     make(map[int64]*model.TaskDTO),
		SnapTasks: make(map[string]*rbody.ScheduledTask),
	}

	controllerUrl := url.URL{Scheme: "ws", Host: *addr, Path: fmt.Sprintf("/socket/%s/%d", *nodeName, Version)}
	conn, err := connect(controllerUrl)
	if err != nil {
		log.Fatalf("unable to connect to server on url %s: %s", controllerUrl.String(), err)
	}

	//create new session, allow 1000 events to be queued in the writeQueue before Emit() blocks.
	sess := session.NewSession(conn, 1000)
	sess.On("disconnect", func() {
		// on disconnect, reconnect.
		ticker := time.NewTicker(time.Second)
		connected := false
		for !connected {
			select {
			case <-shutdownStart:
				ticker.Stop()
				return
			case <-ticker.C:
				conn, err := connect(controllerUrl)
				if err == nil {
					sess.Conn = conn
					connected = true
					go sess.Start()
				}
			}
		}
		ticker.Stop()
	})

	sess.On("heartbeat", func(body []byte) {
		log.Debugf("recieved heartbeat event. %s", body)
	})

	sess.On("taskUpdate", HandleTaskUpdate())
	sess.On("taskAdd", HandleTaskAdd())
	sess.On("taskRemove", HandleTaskRemove())

	go sess.Start()
	//send our MetricCatalog
	body, err := json.Marshal(catalog)
	if err != nil {
		log.Fatal(err)
	}
	e := &message.Event{Event: "catalog", Payload: body}
	sess.Emit(e)

	//periodically send an Updated Catalog.
	go SendCatalog(sess, shutdownStart)

	//wait for interupt Signal.
	<-interrupt
	log.Info("interrupt")
	close(shutdownStart)
	sess.Close()
	return
}

func HandleTaskUpdate() interface{} {
	return func(data []byte) {
		tasks := make([]*model.TaskDTO, 0)
		err := json.Unmarshal(data, &tasks)
		if err != nil {
			log.Errorf("failed to decode taskUpdate payload. %s", err)
			return
		}
		log.Debugf("TaskList. %s", data)
		for _, t := range tasks {
			if err := GlobalTaskCache.AddTask(t); err != nil {
				log.Errorf("failed to add task to cache. %s", err)
			}
		}
	}
}

func HandleTaskAdd() interface{} {
	return func(data []byte) {
		task := model.TaskDTO{}
		err := json.Unmarshal(data, &task)
		if err != nil {
			log.Errorf("failed to decode taskAdd payload. %s", err)
			return
		}
		log.Debugf("Adding Task. %s", data)
		if err := GlobalTaskCache.AddTask(&task); err != nil {
			log.Errorf("failed to add task to cache. %s", err)
		}
	}
}

func HandleTaskRemove() interface{} {
	return func(data []byte) {
		task := model.TaskDTO{}
		err := json.Unmarshal(data, &task)
		if err != nil {
			log.Errorf("failed to decode taskAdd payload. %s", err)
			return
		}
		log.Debugf("Removing Task. %s", data)
		if err := GlobalTaskCache.RemoveTask(&task); err != nil {
			log.Errorf("failed to remove task from cache. %s", err)
		}
	}
}

func IndexTasks(sess *session.Session, shutdownStart chan struct{}) {
	ticker := time.NewTicker(time.Minute * 5)
	for {
		select {
		case <-shutdownStart:
			return
		case <-ticker.C:
			taskList, err := GetSnapTasks()
			if err != nil {
				log.Error(err)
				continue
			}
			if err := GlobalTaskCache.IndexSnapTasks(taskList); err != nil {
				log.Errorf("failed to add task to cache. %s", err)
			}
		}
	}
}

func SendCatalog(sess *session.Session, shutdownStart chan struct{}) {
	ticker := time.NewTicker(time.Minute * 5)
	for {
		select {
		case <-shutdownStart:
			return
		case <-ticker.C:
			catalog, err := GetSnapMetrics()
			if err != nil {
				log.Error(err)
				continue
			}
			body, err := json.Marshal(catalog)
			if err != nil {
				log.Error(err)
				continue
			}
			e := &message.Event{Event: "catalog", Payload: body}
			sess.Emit(e)
		}
	}
}
