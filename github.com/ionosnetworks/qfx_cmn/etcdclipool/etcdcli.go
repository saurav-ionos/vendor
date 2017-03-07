package etcdclipool

import (
	"time"

	clientv3 "github.com/coreos/etcd/clientv3"
	"github.com/ionosnetworks/qfx_cmn/blog"
	"golang.org/x/net/context"
)

type EtcdCli struct {
	EtcdCli *clientv3.Client
	inUse   bool
}

type EtcdClientPool struct {
	connPool chan EtcdCli
	Address  string
}

const (
	ETCDCLIENTTIMEOUT      = 30 * time.Second
	ETCD_CLIENT_POOL_COUNT = 10
)

var (
	etcdCli   *EtcdClientPool
	keyValMap map[string]string
	log       blog.Logger
	ctx       string
)

/*
 All the controller information is stored under /CONTROLLER.
 Client information is split under /CLIENTS and /CLIENTS/CONTROLLER

 /CLIENTS - Will have client Id and its IP address

*/

func Init(etcdAP, context string, logger blog.Logger) {

	log = logger
	ctx = context

	// If ETCD is not provided, we assme that we are in single instance mode.
	// We keep our client data in a map.
	if etcdAP == "" {
		keyValMap = make(map[string]string)
		etcdCli = nil
		return
	}

	log.Debug(ctx, "Using ETCD", blog.Fields{"AP": etcdAP})
	etcdCliLocal := EtcdClientPool{Address: etcdAP, connPool: make(chan EtcdCli, ETCD_CLIENT_POOL_COUNT)}

	for i := 0; i < ETCD_CLIENT_POOL_COUNT; i++ {
		var etcdCli EtcdCli
		if err := etcdCli.init(etcdAP); err == nil {
			etcdCliLocal.connPool <- etcdCli
		}
	}
	etcdCli = &etcdCliLocal
}

func RegisterCallBk(etcdAP, path string, callbk func(interface{}, bool, string, string), cbkctx interface{}) {
	if etcdAP != "" {
		go EtcdV3Monitor(etcdAP, path, callbk, cbkctx)
	}
}

func (cliPool *EtcdClientPool) getCli() EtcdCli {

	cli := <-cliPool.connPool
	return cli
}

func (cliPool *EtcdClientPool) putCli(cli EtcdCli) {

	if cli.EtcdCli == nil {
		log.Crit(ctx, "Releasing invalid client", nil)
	}
	cliPool.connPool <- cli
}

func (cliPool *EtcdClientPool) reconnectCli(cli EtcdCli) error {

	if cli.EtcdCli != nil {
		cli.EtcdCli.Close()
	}
	return cli.init(cliPool.Address)
}

func (cli *EtcdCli) init(etcdAddress string) error {
	dialTimeout := time.Duration(ETCDCLIENTTIMEOUT)
	localCli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdAddress},
		DialTimeout: dialTimeout,
	})
	if err != nil {
		log.Crit(ctx, "Failed to connect to ETCD", blog.Fields{"AP": etcdAddress})
		return err
	}
	cli.EtcdCli = localCli
	cli.inUse = false
	return nil
}

func GetKey(key string) string {

	// t1 := time.Now()
	// defer fmt.Println("Get Key Time taken", time.Since(t1))

	if etcdCli == nil {
		if val, found := keyValMap[key]; found == true {
			return val
		}
	} else {
		if val := EtcdV3Get(key); len(val) != 0 {
			return val[0]
		}
	}
	return ""
}

func SetKeyVal(key, val string, leasetime int) {

	// t1 := time.Now()
	// defer fmt.Println("Set Key Time taken", time.Since(t1))
	if etcdCli == nil {
		keyValMap[key] = val
	} else {
		EtcdV3Set(key, val, int64(leasetime))
	}
}

func DeleteKey(key string) {

	if etcdCli != nil {
		EtcdV3Del(key)
	} else {
		delete(keyValMap, key)
	}
}

func EtcdV3Set(key, value string, timeout int64) {

	var err error

	etcli := etcdCli.getCli()
	defer etcdCli.putCli(etcli)
	cli := etcli.EtcdCli

	if timeout != 0 {
		// minimum lease TTL is 5-second
		ectx, cancel := context.WithTimeout(context.Background(), ETCDCLIENTTIMEOUT)
		resp, err1 := cli.Grant(ectx, timeout)
		if err1 != nil {
			log.Err(ctx, "ETCD grant set Failed", blog.Fields{"key": key, "val": value})
			return
		}

		_, err = cli.Put(context.Background(), key, value, clientv3.WithLease(resp.ID))
		cancel()
	} else {
		_, err = cli.Put(context.Background(), key, value)
		// cancel()
	}

	if err != nil {
		log.Err(ctx, "ETCD value set Failed", blog.Fields{"key": key, "val": value, "err": err.Error()})
	}
}

func EtcdV3Get(key string) []string {

	etcli := etcdCli.getCli()
	defer etcdCli.putCli(etcli)

	cli := etcli.EtcdCli

	requestTimeout := time.Duration(ETCDCLIENTTIMEOUT)
	lctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	resp, err := cli.Get(lctx, key)
	cancel()
	if err != nil {
		log.Err(ctx, "ETCD value get Failed", blog.Fields{"key": key, "err": err.Error()})
	}
	var ret []string
	for _, ev := range resp.Kvs {
		ret = append(ret, string(ev.Value))
	}
	return ret
}

func EtcdV3Del(key string) {

	etcli := etcdCli.getCli()
	defer etcdCli.putCli(etcli)

	cli := etcli.EtcdCli
	// delete the keys
	requestTimeout := time.Duration(ETCDCLIENTTIMEOUT)
	lctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	_, err := cli.Delete(lctx, key, clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Err(ctx, "ETCD value delete Failed", blog.Fields{"key": key, "err": err.Error()})
	}
}

func EtcdV3Monitor(etcdAP, keytoWatch string, callbk func(interface{}, bool, string, string), cbkctx interface{}) {

	dialTimeout := time.Duration(ETCDCLIENTTIMEOUT)
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdAP},
		DialTimeout: dialTimeout,
	})

	//We cannot make any progress if client connection fails.
	if err != nil {
		log.Err(ctx, "ETCD connection failed ", blog.Fields{"err": err.Error()})
		time.Sleep(10 * time.Second)
		panic("ETCD connection failed")
	}
	defer cli.Close()

	log.Info(ctx, "Watching for change", blog.Fields{"key": keytoWatch})

	rch := cli.Watch(context.Background(), keytoWatch, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			bdelete := false
			if ev.Type == clientv3.EventTypeDelete {
				bdelete = true
			}

			callbk(cbkctx, bdelete, string(ev.Kv.Key), string(ev.Kv.Value))
		}
	}
}
