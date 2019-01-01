package url_shortner

import (
	"context"
	"errors"
	"fmt"
	"go.etcd.io/etcd/client"
	"strconv"
	"strings"
	"time"
)

const (
	LOCK   = "on"
	UNLOCK = "off"
)

type EtcdConfig struct {
	Hosts       []string
	ReqTimeout  time.Duration
	NodeId      string
	RootKey     string
	UnlockCheck int

	checkpointKey       string
	rangeKey            string
	rangeIncLockFlagKey string
	rangeIncCounterKey  string
}

type etcdDatasource struct {
	client client.KeysAPI
	cnf    *EtcdConfig
	ctx    context.Context
	locker string
}

func (cnf *EtcdConfig) init() error {
	if len(cnf.Hosts) < 1 {
		return errors.New("hosts does not set")
	}

	if cnf.ReqTimeout == 0 {
		cnf.ReqTimeout = time.Second * 3
	}

	if cnf.RootKey == "" {
		cnf.RootKey = "/url_shortener"
	}

	cnf.checkpointKey = cnf.RootKey + "/checkpoint/" + cnf.NodeId
	cnf.rangeKey = cnf.RootKey + "/range"
	cnf.rangeIncLockFlagKey = cnf.rangeKey + "/locker"
	cnf.rangeIncCounterKey = cnf.rangeKey + "/counter"

	return nil
}

func NewEtcd(cnf *EtcdConfig) (*etcdDatasource, error) {
	if err := cnf.init(); err != nil {
		return nil, err
	}

	cli, err := client.New(client.Config{
		Endpoints:               cnf.Hosts,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: cnf.ReqTimeout,
	})

	if err != nil {
		return nil, err
	}

	etcd := &etcdDatasource{
		client: client.NewKeysAPI(cli),
		cnf:    cnf,
		ctx:    context.Background(),
	}

	etcd.initBasicKeys()

	return etcd, etcd.checkCriticalRequirements()
}

func (e *etcdDatasource) initBasicKeys() {
	var err error
	if _, err = e.client.Get(e.ctx, e.cnf.rangeIncLockFlagKey, nil); err != nil {
		e.client.Set(e.ctx, e.cnf.rangeIncLockFlagKey, UNLOCK, nil)
	}

	go e.watchOnLocker()

	if _, err = e.client.Get(e.ctx, e.cnf.rangeIncCounterKey, nil); err != nil {
		e.client.Set(e.ctx, e.cnf.rangeIncCounterKey, "0", nil)
	}
}

func (e *etcdDatasource) watchOnLocker() {
	watcher := e.client.Watcher(e.cnf.rangeIncLockFlagKey, &client.WatcherOptions{})

	for true {
		res, err := watcher.Next(context.Background())
		if err != nil {
			println(err.Error())
		} else {
			e.locker = res.Node.Value
		}
	}
}

func (e *etcdDatasource) checkCriticalRequirements() (err error) {
	_, err = e.client.Get(e.ctx, e.cnf.rangeKey, nil)
	return
}

func (e *etcdDatasource) getRestoreRange() (int, int, error) {
	data, err := e.client.Get(e.ctx, e.cnf.checkpointKey, nil)
	if err != nil {
		return e.getNextRange()
	}

	return valueSplitter(data.Node.Value)
}

func (e *etcdDatasource) getNextRange() (start int, end int, err error) {
	for count := 0; count < e.cnf.UnlockCheck && e.locker != UNLOCK; count++ {
		println("the counter is locked; sleep 1 sec to recheck again")
		time.Sleep(time.Second)
	}

	// lock access
	if _, err = e.client.Update(e.ctx, e.cnf.rangeIncLockFlagKey, LOCK); err != nil {
		return
	}

	// increment key
	strCounter, _ := e.client.Get(e.ctx, e.cnf.rangeIncCounterKey, nil)
	counter, _ := strconv.Atoi(strCounter.Node.Value)
	counter++
	e.client.Set(e.ctx, e.cnf.rangeIncCounterKey, strconv.Itoa(counter), nil)

	// fetch range based on counter
	rangeStr, err := e.client.Get(e.ctx, fmt.Sprintf("%s/%d", e.cnf.rangeKey, counter), nil)
	if err != nil {
		panic(err.Error())
	}

	start, end, err = valueSplitter(rangeStr.Node.Value)
	e.commit(start, end)

	// release the locker
	if _, err = e.client.Update(e.ctx, e.cnf.rangeIncLockFlagKey, UNLOCK); err != nil {
		return
	}

	return
}

// to improve performance on high load you can commit changes after "n" change
// if you want to save count number after "n" change you must add +n to restore counter
func (e *etcdDatasource) commit(counter int, end int) error {
	_, err := e.client.Set(e.ctx, e.cnf.checkpointKey, fmt.Sprintf("%d-%d", counter, end), nil)
	return err
}

// watch on lock
// register service

func valueSplitter(value string) (int, int, error) {
	numbers := strings.Split(value, "-")

	startFrom, err := strconv.Atoi(numbers[0])
	if err != nil {
		return 0, 0, err
	}

	maxNumber, err := strconv.Atoi(numbers[1])
	if err != nil {
		return 0, 0, err
	}

	return startFrom, maxNumber, nil
}
