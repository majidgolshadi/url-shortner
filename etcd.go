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
	Hosts      []string
	ReqTimeout time.Duration
	NodeId     string
	RootKey    string

	checkpointKey       string
	rangeKey            string
	rangeIncLockFlagKey string
	rangeIncCounterKey  string
}

type etcdDatasource struct {
	client client.KeysAPI
	cnf    *EtcdConfig
	ctx    context.Context
	sleepCount int
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

	// create locker if not exist
	// create counter if not exist

	etcd :=  &etcdDatasource{
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

	if _, err = e.client.Get(e.ctx, e.cnf.rangeIncCounterKey, nil); err != nil {
		e.client.Set(e.ctx, e.cnf.rangeIncCounterKey, "0", nil)
	}
}

func (e *etcdDatasource) checkCriticalRequirements() (err error) {
	_, err = e.client.Get(e.ctx, e.cnf.rangeKey, nil)
	return
}

func (e *etcdDatasource) restoreStartPoint() (int, int, error) {
	data, err := e.client.Get(e.ctx, e.cnf.checkpointKey, nil)
	if err != nil {
		return e.getNewRange()
	}

	return valueSplitter(data.Node.Value)
}

func (e *etcdDatasource) getNewRange() (start int, end int, err error) {
check:
	data, _ := e.client.Get(e.ctx, e.cnf.rangeIncLockFlagKey, nil)
	if data.Node.Value != UNLOCK {
		time.Sleep(time.Second)
		e.sleepCount++
		println("the counter is locked so sleep 1 sec to recheck after")


		// restore service after 5 second
		if e.sleepCount > 5 {
			e.sleepCount = 0
			e.client.Update(e.ctx, e.cnf.rangeIncLockFlagKey, UNLOCK)
		}

		goto check
	}

	if _, err = e.client.Update(e.ctx, e.cnf.rangeIncLockFlagKey, LOCK); err != nil {
		return
	}

	// increment key
	strCounter, _ := e.client.Get(e.ctx, e.cnf.rangeIncCounterKey, nil)
	counter, _ := strconv.Atoi(strCounter.Node.Value)
	counter++
	e.client.Set(e.ctx, e.cnf.rangeIncCounterKey, strconv.Itoa(counter), nil)

	// fetch range
	rangeStr, err := e.client.Get(e.ctx, fmt.Sprintf("%s/%d", e.cnf.rangeKey, counter), nil)
	if err != nil {
		panic(err.Error())
	}

	// init return numbers
	start, end, err = valueSplitter(rangeStr.Node.Value)
	e.save(start, end)

	if _, err = e.client.Update(e.ctx, e.cnf.rangeIncLockFlagKey, UNLOCK); err != nil {
		return
	}

	return
}

// to improve performance on high load you can commit changes after "n" change
// if you want to save count number after "n" change you must add +n to restore counter
func (e *etcdDatasource) save(counter int, end int) {
	_, err := e.client.Set(e.ctx, e.cnf.checkpointKey, fmt.Sprintf("%d-%d", counter, end), nil)
	if err != nil {
		println(err.Error())
	}
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
