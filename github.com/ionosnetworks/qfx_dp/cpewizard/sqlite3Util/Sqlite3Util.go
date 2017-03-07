package sqlite3Util

import (
	"database/sql"
	"fmt"
	DataObj "github.com/ionosnetworks/qfx_dp/cpewizard/DataObjects"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

var (
	db *sql.DB
)

func CreateSqliteDB() {
	db = InitDB("cpeData.db")
	/*CreateTable()
	ClearTable()*/
	fmt.Println("Database Tables created!")
}

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	db.SetMaxOpenConns(1)
	if err != nil {
		panic(err)
	}
	if db == nil {
		panic("db nil")
	}
	return db
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func ClearTable() {
	sql_delete_table :=
		`DELETE FROM file_info;`
	_, err := db.Exec(sql_delete_table)
	checkErr(err)
	sql_delete_table =
		`DELETE FROM path_info;`
	_, err = db.Exec(sql_delete_table)
	checkErr(err)
}

func CheckLastProcess() string {
	sqlReadLastStatus := `
	select Status from log_info where (EndTime) IN (select max(EndTime) from log_info);
	`
	rows, err := db.Query(sqlReadLastStatus)
	checkErr(err)
	defer rows.Close()

	var status string
	for rows.Next() {
		err = rows.Scan(&status)
		checkErr(err)
	}
	return status
}

func CreateTable() {
	// create tables if not exists
	sql_table := `
	 PRAGMA synchronous = NORMAL;
	    PRAGMA journal_mode = WAL;
	    PRAGMA wal_autocheckpoint = 16384;
	    PRAGMA auto_vacuum = FULL;	 	       
	CREATE TABLE IF NOT EXISTS file_info(
		PathId INTEGER,
		FileName BLOB,
		FileSize INTEGER,
		IsDir BOOLEAN,
		IsExported BOOLEAN,
		Level INTEGER,
		ModTime DATE
	);
	`
	_, err := db.Exec(sql_table)
	checkErr(err)

	sql_table = `
	    PRAGMA synchronous = NORMAL;
	    PRAGMA journal_mode = WAL;
	    PRAGMA wal_autocheckpoint = 16384;
	    PRAGMA auto_vacuum = FULL;  	   
	CREATE TABLE IF NOT EXISTS path_info(
		Id INTEGER primary key AUTOINCREMENT,
		FilePath TEXT UNIQUE
	);
	`
	_, err = db.Exec(sql_table)
	checkErr(err)

	sql_table = `
	    PRAGMA synchronous = NORMAL;
	    PRAGMA journal_mode = WAL;
	    PRAGMA wal_autocheckpoint = 16384;
	    PRAGMA auto_vacuum = FULL;	   
	CREATE TABLE IF NOT EXISTS log_info(
		StartTime DATETIME,
		EndTime DATETIME,
		FilePath TEXT,
		Status TEXT
	);
	`
	_, err = db.Exec(sql_table)
	checkErr(err)

	sql_index := `CREATE INDEX PathIdIndex on file_info(PathId);`
	_, err = db.Exec(sql_index)
	checkErr(err)
}

func StoreLogInfo(startTime time.Time, endTime time.Time, status string) {
	sql_LogInfo := `
		INSERT INTO log_info(StartTime,EndTime,Status) VALUES(?,?,?)`
	LogInfoStmt, err := db.Prepare(sql_LogInfo)
	checkErr(err)
	defer LogInfoStmt.Close()
	tx, err := db.Begin()
	checkErr(err)
	_, err = tx.Stmt(LogInfoStmt).Exec(startTime, endTime, status)
	if err != nil {
		checkErr(err)
		fmt.Println("doing rollback")
		tx.Rollback()
	} else {
		tx.Commit()
	}
}

func UpdateLogInfo(startTime time.Time, endTime time.Time, status string) {
	sql_LogInfo := `UPDATE log_info SET EndTime=?,Status=? WHERE StartTime=?`

	LogInfoStmt, err := db.Prepare(sql_LogInfo)
	checkErr(err)
	defer LogInfoStmt.Close()
	tx, err := db.Begin()
	checkErr(err)
	_, err = tx.Stmt(LogInfoStmt).Exec(endTime, status, startTime)
	if err != nil {
		checkErr(err)
		fmt.Println("doing rollback")
		tx.Rollback()
	} else {
		tx.Commit()
	}
}

/*
	Saving by appending
*/
func StoreFileList(fileList []DataObj.File) {
	sql_PathInfo := `
		INSERT OR IGNORE INTO path_info(Id,FilePath) VALUES`
	sql_FileInfo := `
	INSERT OR REPLACE INTO file_info(
		PathId,
		FileName,
		FileSize,
		IsDir,
		IsExported,
		Level,
		ModTime
	) VALUES `

	valsPath := []interface{}{}
	valsFile := []interface{}{}
	for _, file := range fileList {
		sql_PathInfo += "(?,?),"
		sql_FileInfo += "((SELECT Id FROM path_info WHERE FilePath = ?),?, ?, ?, ?, ?, ?),"
		valsPath = append(valsPath, nil, file.FilePath)
		valsFile = append(valsFile, file.FilePath, file.FileName, file.FileSize,
			file.IsDir, file.IsExported, file.Level, file.ModTime)
	}

	sql_PathInfo = sql_PathInfo[0 : len(sql_PathInfo)-1]
	sql_FileInfo = sql_FileInfo[0 : len(sql_FileInfo)-1]

	PathInfoStmt, err := db.Prepare(sql_PathInfo)
	checkErr(err)
	fileInfoStmt, err := db.Prepare(sql_FileInfo)
	checkErr(err)
	defer PathInfoStmt.Close()
	defer fileInfoStmt.Close()

	tx, err := db.Begin()
	checkErr(err)
	_, err = tx.Stmt(PathInfoStmt).Exec(valsPath...)
	_, err1 := tx.Stmt(fileInfoStmt).Exec(valsFile...)
	if err != nil || err1 != nil {
		checkErr(err)
		checkErr(err1)
		fmt.Println("doing rollback")
		tx.Rollback()
	} else {
		tx.Commit()
	}
}

/*
Saving by iterating within transaction
*/
func StoreFileInfoList(fileList []DataObj.File) {
	sql_PathInfo := `
		INSERT OR IGNORE INTO path_info(Id,FilePath) VALUES(?,?)`

	sql_FileInfo := `
	INSERT OR REPLACE INTO file_info(
		PathId,
		FileName,
		FileSize,
		IsDir,
		IsExported,
		Level,
		ModTime
	) values((SELECT Id FROM path_info WHERE FilePath = ?),?, ?, ?, ?, ?, ?)
	`

	tx, err := db.Begin()
	checkErr(err)
	PathInfoStmt, err := db.Prepare(sql_PathInfo)
	checkErr(err)

	fileInfoStmt, err := db.Prepare(sql_FileInfo)
	checkErr(err)
	defer PathInfoStmt.Close()
	defer fileInfoStmt.Close()

	for _, file := range fileList {
		_, err1 := tx.Stmt(PathInfoStmt).Exec(nil, file.FilePath)

		_, err2 := tx.Stmt(fileInfoStmt).Exec(file.FilePath, file.FileName, file.FileSize, file.IsDir, file.IsExported, file.Level, file.ModTime)
		if err1 != nil || err2 != nil {
			checkErr(err1)
			checkErr(err2)
			fmt.Println("doing rollback")
			tx.Rollback()
		}
	}
	tx.Commit()
}

//To Read the DB Root Dirs
func GetRootDirs(pathId int) []DataObj.File {
	var result []DataObj.File
	sql_read := `
		SELECT FILE_INFO.FileName			
	FROM FILE_INFO
	WHERE FILE_INFO.PathId = ?;
	`
	stm, err := db.Prepare(sql_read)
	checkErr(err)
	defer stm.Close()
	rows, err := stm.Query(pathId)
	checkErr(err)
	defer rows.Close()
	for rows.Next() {
		file := DataObj.File{}
		err := rows.Scan(&file.FileName)
		checkErr(err)
		result = append(result, file)
	}
	return result
}

//Get Total file count for given path
func GetTotalDirFileCount(pathId int) int {
	var count int
	sql_read := `SELECT count(*) FROM FILE_INFO where PathId = ?`
	stm, err := db.Prepare(sql_read)
	checkErr(err)
	defer stm.Close()
	rows, err := stm.Query(pathId)
	checkErr(err)
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&count)
		checkErr(err)
	}
	fmt.Println("Got count of files", count)
	return count
}

//Get Total Search file count for given path
func GetTotalSearchCount(searchPattern string, dirFullPath string) int {
	var count int
	sql_read := `SELECT count(*) FROM FILE_INFO F, PATH_INFO P where F.PathId=P.Id and FileName LIKE ? and P.FilePath LIKE ?;`
	stm, err := db.Prepare(sql_read)
	checkErr(err)
	defer stm.Close()
	rows, err := stm.Query(("%" + searchPattern + "%"), (dirFullPath + "%"))
	checkErr(err)
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&count)
		checkErr(err)
	}
	fmt.Println("Got count of search", count)
	return count
}

func GetPathId(dirFullPath string) int {
	fmt.Println("***", dirFullPath)
	var pathId int
	sql_read := `SELECT Id FROM PATH_INFO WHERE FilePath= ?;`
	stm, err := db.Prepare(sql_read)
	checkErr(err)
	defer stm.Close()
	rows, err := stm.Query(dirFullPath)
	checkErr(err)
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&pathId)
		checkErr(err)
	}
	fmt.Println("**", pathId)
	return pathId
}

func GetFilePath(pathId int) string {
	var dirFullPath string
	sql_read := `SELECT FilePath FROM PATH_INFO WHERE Id= ?;`
	stm, err := db.Prepare(sql_read)
	checkErr(err)
	defer stm.Close()
	rows, err := stm.Query(pathId)
	checkErr(err)
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&dirFullPath)
		checkErr(err)
	}
	return dirFullPath
}

//To Read the DB Root Dirs
func GetDirInfo(pathId int, dirFullPath string, pageNum int, searchPattern string) []DataObj.File {
	var result []DataObj.File
	var sql_read string
	var stm *sql.Stmt
	var err error
	var rows *sql.Rows
	fmt.Println("Getting dir info for: pathId ", pathId, " pageNum ", pageNum, "searchPattern ", searchPattern)
	if searchPattern != "" {
		sql_read = `
		SELECT F.PathId,
		    F.FileName,
			F.FileSize,
			F.IsDir,
			F.IsExported,
			F.ModTime
	FROM FILE_INFO F, PATH_INFO P
	WHERE F.PathId=P.Id and F.FileName LIKE ? and P.FilePath LIKE ? ORDER BY F.FileName LIMIT 100 OFFSET ? ;
	`
		stm, err = db.Prepare(sql_read)
		checkErr(err)
		defer stm.Close()
		rows, err = stm.Query(("%" + searchPattern + "%"), (dirFullPath + "%"), pageNum)
		checkErr(err)
	} else {
		sql_read = `
		SELECT FILE_INFO.PathId, 
		    FILE_INFO.FileName,
			FILE_INFO.FileSize,
			FILE_INFO.IsDir,
			FILE_INFO.IsExported,			
			FILE_INFO.ModTime
	FROM FILE_INFO
	WHERE PathId = ? ORDER BY FileName LIMIT 100 OFFSET ? ;
	`
		stm, err = db.Prepare(sql_read)
		checkErr(err)
		defer stm.Close()
		rows, err = stm.Query(pathId, pageNum)
		checkErr(err)
	}

	defer rows.Close()
	for rows.Next() {
		file := DataObj.File{}
		err := rows.Scan(&file.PathId, &file.FileName, &file.FileSize, &file.IsDir,
			&file.IsExported, &file.ModTime)
		checkErr(err)
		result = append(result, file)
	}

	for index := range result {
		result[index].FilePath = GetFilePath(result[index].PathId)
	}

	return result
}

func ClearExistingData(dirFullPath string) {
	fmt.Println("Clear: ", dirFullPath)
	sql_read := `DELETE FROM FILE_INFO WHERE PathId IN(SELECT Id from PATH_INFO where FilePath like ?);`
	stm, err := db.Prepare(sql_read)
	checkErr(err)
	defer stm.Close()
	_, err = stm.Exec(dirFullPath + "%")
	checkErr(err)
	sql_read = `DELETE FROM PATH_INFO WHERE FilePath like ?;`
	stm, err = db.Prepare(sql_read)
	checkErr(err)
	defer stm.Close()
	_, err = stm.Exec(dirFullPath + "%")
	checkErr(err)
	fmt.Println("Cleared: ", dirFullPath)
}
