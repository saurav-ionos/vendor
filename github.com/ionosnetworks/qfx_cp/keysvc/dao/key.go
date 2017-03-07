package dao

import (
	"os"

	"github.com/ionosnetworks/qfx_cmn/blog"
	kr "github.com/ionosnetworks/qfx_cmn/keyreader"
	o "github.com/ionosnetworks/qfx_cp/keysvc/objects"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type (
	DAOErr             int
	DaoKeyCreateIn     o.KeyCreateIn
	DaoKeyIn           o.KeyIn
	DaoClientKeyIn     o.ClientKeyIn
	DaoKeyOut          o.KeyOut
	DaoKeyEntryOut     o.KeyEntryOut
	DaoKeyEntryOutList o.KeyEntryOutList
	DaoKeyCredential   o.KeyCredential
)

const (
	DAO_OK  DAOErr = 1
	DAO_ERR DAOErr = 0

	dbSvr      = "127.0.0.1:27017"
	Username   = ""
	Password   = ""
	Database   = "lft2_0"
	Collection = "lft_key"
)

type DbClient struct {
	session *mgo.Session
	inUse   bool
}

type DbClientPool struct {
	connPool chan DbClient
	Address  string
}

var (
	dbCli                 *DbClientPool
	logger                blog.Logger
	ctx                   = ""
	DEFAULT_API_CREDS     = []string{"/"}
	DEFAULT_FEATURE_CREDS = []string{"log"}
)

const (
	DB_CLIENT_POOL_COUNT   = 10
	MASTER_IONOS_KEY_NAME  = "MASTER_IONOS_KEY"
	MASTER_KEY_TENANT_NAME = "IONOS"
	MASTER_KEY_CLIENT_NAME = "IONOS"
	MASTER_KEY_PATH        = "/masterkeys/keyfile"
)

func InitDao(context string, log blog.Logger) {
	dbSvr := "127.0.0.1:27017"
	if val := os.Getenv("KEY_DB_SERVER"); val != "" {
		dbSvr = val
	}

	ctx = context
	logger = log

	logger.Debug(ctx, "Using DB", blog.Fields{"AP": dbSvr})
	dbCliLocal := DbClientPool{Address: dbSvr, connPool: make(chan DbClient, DB_CLIENT_POOL_COUNT)}

	for i := 0; i < DB_CLIENT_POOL_COUNT; i++ {
		var dbCli DbClient
		if err := dbCli.init(dbSvr); err == nil {
			dbCliLocal.connPool <- dbCli
		}
	}
	dbCli = &dbCliLocal
}

func checkAndCreateMasterKey() {
	// Check for Master IONOS key. If does not exists, create it.

	masterKey := DaoClientKeyIn{Name: MASTER_IONOS_KEY_NAME,
		TenantName: MASTER_KEY_TENANT_NAME, ClientName: MASTER_KEY_CLIENT_NAME}

	entrylist, err := masterKey.DAOLoad()

	if err != DAO_OK || len(entrylist.KeyIdList) == 0 {

		accessKey := kr.New(MASTER_KEY_PATH)
		// Create the master key.
		ionosKey := DaoKeyCreateIn{Name: MASTER_IONOS_KEY_NAME,
			TenantName: MASTER_KEY_TENANT_NAME, ClientName: MASTER_KEY_CLIENT_NAME,
			AccessKey: accessKey.Key, AccessSecret: accessKey.Secret,
			Credential: o.KeyCredential{ApiList: DEFAULT_API_CREDS, FeatureList: DEFAULT_FEATURE_CREDS}}

		if err := ionosKey.DAOSave(); err != DAO_OK {
			logger.Debug(ctx, "Failed to save the master key", nil)
		} else {
			logger.Debug(ctx, "Master key saved", nil)
		}
	} else {
		logger.Debug(ctx, "Master key exists", nil)
	}
}

func (cliPool *DbClientPool) getCli() DbClient {

	cli := <-cliPool.connPool
	return cli
}

func (cliPool *DbClientPool) putCli(cli DbClient) {

	if cli.session == nil {
		logger.Crit(ctx, "Releasing invalid client", nil)
	}
	cliPool.connPool <- cli
}

func (cli *DbClient) init(dbAddress string) error {

	session, err := mgo.Dial(dbAddress)

	if err != nil {
		logger.Crit(ctx, "Failed to connect to DB", blog.Fields{"AP": dbAddress})
		return err
	}
	cli.session = session
	cli.inUse = false
	return nil
}

func SetLogger(context string, log blog.Logger) {
	ctx = context
	logger = log
}

func (key DaoKeyCreateIn) DAOSave() DAOErr {

	client := dbCli.getCli()
	defer dbCli.putCli(client)

	session := client.session

	coll := session.DB(Database).C(Collection)

	if err := coll.Insert(key); err != nil {
		panic(err)
	}

	return DAO_OK
}

func (key DaoClientKeyIn) DAOLoad() (DaoKeyEntryOutList, DAOErr) {

	client := dbCli.getCli()
	defer dbCli.putCli(client)

	session := client.session

	coll := session.DB(Database).C(Collection)

	keylist := make([]string, 0)
	keyentry := DaoKeyCreateIn{}
	iter := coll.Find(bson.M{"name": key.Name, "tenantname": key.TenantName,
		"clientname": key.ClientName}).Iter()

	for iter.Next(&key) {
		keylist = append(keylist, keyentry.AccessKey)
	}

	if err := iter.Close(); err != nil {
		logger.Err(ctx, "Internal error", blog.Fields{"Key": key.TenantName})
		return DaoKeyEntryOutList{}, DAO_ERR
	}

	return DaoKeyEntryOutList{KeyIdList: keylist, ErrCode: "OK"}, DAO_OK

}

// Retrieve details of key given its AccessKey
func (key DaoKeyIn) DAOLoad() (DaoKeyEntryOut, DAOErr) {

	client := dbCli.getCli()
	defer dbCli.putCli(client)

	session := client.session

	coll := session.DB(Database).C(Collection)
	keyOut := DaoKeyCreateIn{}
	if err := coll.Find(bson.M{"accesskey": key.AccessKey}).One(&keyOut); err != nil {
		logger.Err(ctx, "Key not found", blog.Fields{"Key": key.AccessKey})
	} else {
		return DaoKeyEntryOut{Name: keyOut.Name, TenantName: keyOut.TenantName,
			ClientName: keyOut.ClientName, AccessKey: keyOut.AccessKey,
			AccessSecret: keyOut.AccessSecret, Credential: keyOut.Credential, ErrCode: "OK"}, DAO_OK
	}
	return DaoKeyEntryOut{}, DAO_ERR
}

func (key DaoKeyIn) DAODelete() (o.ObjectDeleteOut, DAOErr) {

	client := dbCli.getCli()
	defer dbCli.putCli(client)

	session := client.session

	coll := session.DB(Database).C(Collection)
	if _, err := coll.RemoveAll(bson.M{"accesskey": key.AccessKey}); err != nil {
		logger.Err(ctx, "Key delete failed", blog.Fields{"Key": key.AccessKey})
	} else {
		logger.Info(ctx, "Key delete done", blog.Fields{"Key": key.AccessKey})
	}

	return o.ObjectDeleteOut{"OK"}, DAO_OK
}
