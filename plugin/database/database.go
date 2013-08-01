package database

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"path/filepath"
	"sync"
)

type dbkey struct {
	driver string
	path   string
}

type dbent struct {
	db       *sql.DB
	revdb    *sql.DB
	openerr  error
	closeerr error
	count    uint
}

var dbs map[dbkey]dbent
var revdbs map[*sql.DB]dbkey
var mutex sync.Mutex

func init() {
	dbs = make(map[dbkey]dbent)
	revdbs = make(map[*sql.DB]dbkey)
}

// Returns the specified database, opening it if necessary.
// If opening the database fails, this function will return an error,
// and will not re-try opening the database unless you Clear() it.
//
// The returned database is ref-counted; you must call Close() with
// the same driver/path combo to balance every successful Open().
// You should not call Close() if the Open() failed.
func Open(driver, path string) (*sql.DB, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if newpath, err := filepath.Abs(path); err != nil {
		return nil, err
	} else {
		path = newpath
	}
	key := dbkey{driver: driver, path: path}

	if dbent, ok := dbs[key]; ok {
		if dbent.openerr != nil {
			return nil, dbent.openerr
		}
		if dbent.db != nil {
			dbent.count += 1
			dbs[key] = dbent
			return dbent.db, nil
		}
		// otherwise we had a closeerr
	}

	db, err := sql.Open(driver, path)
	dbs[key] = dbent{db: db, revdb: db, openerr: err, count: 1}
	if db != nil {
		revdbs[db] = key
	}
	return db, err
}

// Closes the specified database (if this is the last reference to it).
// If a previous Close returned an error, subsequent Closes will continue
// to return the error until either Open() or Clear() is called.
func Close(db *sql.DB) error {
	if db == nil {
		return errors.New("error: database.Close() may not be called with a nil argument")
	}
	mutex.Lock()
	defer mutex.Unlock()

	if key, ok := revdbs[db]; ok {
		dbent := dbs[key]
		if dbent.closeerr != nil {
			return dbent.closeerr
		}
		if dbent.db != nil {
			dbent.count -= 1
			if dbent.count == 0 {
				dbent.closeerr = dbent.db.Close()
				dbent.db = nil
				if dbent.closeerr == nil {
					delete(dbs, key)
					delete(revdbs, dbent.revdb)
					return nil
				}
			}
			dbs[key] = dbent
			return dbent.closeerr
		}
		// otherwise must be an openerr
	}

	return errors.New("error: No open database with that driver/path")
}

// Clears any cached open/close errors.
// Returns an error if the database is currently open.
func Clear(driver, path string) error {
	mutex.Lock()
	defer mutex.Unlock()

	if newpath, err := filepath.Abs(path); err != nil {
		return err
	} else {
		path = newpath
	}
	key := dbkey{driver: driver, path: path}

	if dbent, ok := dbs[key]; ok {
		if dbent.db != nil {
			return errors.New("error: A database with that driver/path is currently open")
		}
		delete(dbs, key)
		delete(revdbs, dbent.revdb)
	}

	return nil
}
