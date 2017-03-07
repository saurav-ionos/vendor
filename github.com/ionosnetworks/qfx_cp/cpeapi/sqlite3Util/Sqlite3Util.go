package sqlite3Util

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

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

func ClearTable(db *sql.DB) {
	sql_delete_table :=
		`DELETE FROM file_info;`
	_, err := db.Exec(sql_delete_table)
	checkErr(err)
	sql_delete_table =
		`DELETE FROM path_info;`
	_, err = db.Exec(sql_delete_table)
	checkErr(err)
}

func CheckLastProcess(db *sql.DB) string {
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

func CreateTable(db *sql.DB) {
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
		Status TEXT
	);
	`
	_, err = db.Exec(sql_table)
	checkErr(err)
}

func StoreLogInfo(db *sql.DB, startTime time.Time, endTime time.Time, status string) {
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

func UpdateLogInfo(db *sql.DB, startTime time.Time, endTime time.Time, status string) {
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
func StoreFileList(db *sql.DB, fileList []File) {
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
func StoreFileInfoList(db *sql.DB, fileList []File) {
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
func GetRootDirs(db *sql.DB) []File {
	var result []File
	sql_read := `
		SELECT PATH_INFO.FilePath as filePath ,FILE_INFO.FileName,
			FILE_INFO.FileSize,
			FILE_INFO.IsDir,
			FILE_INFO.IsExported,			
			FILE_INFO.ModTime
	FROM FILE_INFO, PATH_INFO
	WHERE FILE_INFO.PathId = PATH_INFO.Id and FILE_INFO.Level=5 and FILE_INFO.IsDir = 1;
		`
	rows, err := db.Query(sql_read)
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		file := File{}
		err := rows.Scan(&file.FilePath, &file.FileName, &file.FileSize, &file.IsDir,
			&file.IsExported, &file.ModTime)
		checkErr(err)
		result = append(result, file)
	}
	return result
}

//Get Total file count for given path
func GetDirCount(db *sql.DB, dirFullPath string) int {
	var count int
	sql_read := `SELECT count(*) FROM FILE_INFO, PATH_INFO 
		WHERE FILE_INFO.PathId = PATH_INFO.Id and 
		PATH_INFO.FilePath = ?`
	stm, err := db.Prepare(sql_read)
	checkErr(err)
	defer stm.Close()
	rows, err := stm.Query(dirFullPath)
	checkErr(err)
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&count)
		checkErr(err)
	}
	return count
}

//To Read the DB Root Dirs
func GetDirInfo(db *sql.DB) []File {
	var result []File
	sql_read := `
		SELECT PATH_INFO.FilePath as filePath ,FILE_INFO.FileName,
			FILE_INFO.FileSize,
			FILE_INFO.IsDir,
			FILE_INFO.IsExported,			
			FILE_INFO.ModTime
	FROM FILE_INFO, PATH_INFO
	WHERE FILE_INFO.PathId = PATH_INFO.Id and FILE_INFO.Level=5 and FILE_INFO.IsDir = 1;
		`
	rows, err := db.Query(sql_read)
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		file := File{}
		err := rows.Scan(&file.FilePath, &file.FileName, &file.FileSize, &file.IsDir,
			&file.IsExported, &file.ModTime)
		checkErr(err)
		result = append(result, file)
	}
	return result
}
