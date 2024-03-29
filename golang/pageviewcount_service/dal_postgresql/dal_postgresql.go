package dal_postgresql

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

// General Functions
func stringIn(val string, valid []string) (rv bool) {
	for _, v := range valid {
		if v == val {
			return true
		}
	}
	return false
}

// Postgres Config object and functions
type PostgresCfg struct {
	hostname string
	username string
	password string
	port     uint16
	database string
	sslmode  string
}

func NewPostgresCfg(loginDetails map[string]interface{}) (pCFG *PostgresCfg) {
	pCFG = &PostgresCfg{hostname: "localhost", port: 5432, sslmode: "disable"}

	for k, v := range loginDetails {
		if k == "type" {
			continue
		}

		switch k {
		case "hostname":
			pCFG.hostname = v.(string)
		case "username":
			pCFG.username = v.(string)
		case "password":
			pCFG.password = v.(string)
		case "database":
			pCFG.database = v.(string)
		case "port":
			pCFG.port = uint16(v.(float64))
		case "ssl":
			var mode string
			if reflect.TypeOf(v).Kind() == reflect.Bool {
				if v.(bool) == true {
					mode = "require"
				} else {
					mode = "disable"
				}
			} else if reflect.TypeOf(v).Kind() == reflect.String {
				// Not implemented yet
				switch v.(string) {
				case "require", "disable":
					mode = v.(string)
				default:
					mode = "verify-full"
				}
			}
			if mode != "" {
				pCFG.sslmode = mode
			}
		}
	}

	return pCFG
}

func (pCFG *PostgresCfg) GetURI() (rv string) {
	args := []string{}

	rv = "postgres://"
	if pCFG.username != "" {
		rv += pCFG.username
		if pCFG.password != "" {
			rv += ":" + pCFG.password
		}
		rv += "@"
	}

	if pCFG.hostname != "" {
		rv += pCFG.hostname
	}
	if pCFG.port > 0 {
		rv += ":" + strconv.FormatUint(uint64(pCFG.port), 10)
	}
	if pCFG.database != "" {
		rv += "/" + pCFG.database
	}
	if pCFG.sslmode != "" {
		args = append(args, "sslmode="+pCFG.sslmode)
	}

	if len(args) > 0 {
		rv += "?" + strings.Join(args, "&")
	}
	return rv
}

type postgresFunctionExecutor struct {
	NumAffectedLastOp int64
	Record            []interface{}

	dbh               *sql.DB
	functionName      string
	functionArguments []interface{}
	dmlArguments      []interface{}
	dmlStatement      string
}

func NewPostgresFunctionExecutor(dbh *sql.DB, functionName string, functionArguments []interface{}) (pFE postgresFunctionExecutor, err error) {
	if dbh == nil {
		return pFE, errors.New("Invalid database handle provided")
	}

	pFE = postgresFunctionExecutor{}
	pFE.dbh = dbh
	pFE.functionName = functionName
	pFE.functionArguments = functionArguments

	return pFE, err
}

func (pFE *postgresFunctionExecutor) ExecuteProc(procArgs map[string]interface{}, args ...string) (err error) {
	statement := bytes.NewBufferString("select * from " + pFE.functionName + "(")

	placeholders := []string{}
	placeholderCounter := int64(1)

	for _, fieldName := range pFE.functionArguments {
		placeholders = append(placeholders, "$"+strconv.FormatInt(placeholderCounter, 10))
		if v, ok := procArgs[fieldName.(string)]; ok {
			pFE.dmlArguments = append(pFE.dmlArguments, v)
		} else {
			pFE.dmlArguments = append(pFE.dmlArguments, nil)
		}
		placeholderCounter += 1
	}

	if placeholderCounter > 0 {
		statement.WriteString(strings.Join(placeholders, ","))
	}
	statement.WriteString(")")

	pFE.dmlStatement = statement.String()

	rows, err := pFE.dbh.Query(pFE.dmlStatement, pFE.dmlArguments...)
	if err != nil {
		return err
	}

	var row map[string]interface{}
	columns, _ := rows.Columns()
	count := len(columns)

	counter := int64(0)
	for rows.Next() {
		values := make([]interface{}, count)
		valuePtrs := make([]interface{}, count)
		for i, _ := range columns {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)

		row = make(map[string]interface{})

		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}

			row[col] = v
		}
		counter += 1

		pFE.Record = append(pFE.Record, row)
	}

	pFE.NumAffectedLastOp = counter

	return err
}

// Postgres Database Handle object and functions
type postgresDataHandler struct {
	Id                map[string]interface{}
	NumAffectedLastOp int64
	RecordNextIdx     map[string]interface{}
	RecordLastOp      map[string]interface{}
	Record            []interface{}

	dbh            *sql.DB
	tableName      string
	batchSize      int64
	EnforceBatch   bool // Capitalized means a 'public' or 'exported' field
	projection     map[string]interface{}
	searchCriteria map[string]interface{}
	sortDirection  string
	primaryKey     []string
	dmlType        string
	dmlStatement   string
	dmlArguments   []interface{}
}

func GetDatabaseHandle(loginDetails map[string]interface{}) (dbh *sql.DB, err error) {
	var connDetailString string

	if hn, ok := loginDetails["hostname"]; ok && strings.HasPrefix(hn.(string), "postgresql://") {
		connDetailString = hn.(string)
		dbh, err = sql.Open("postgres", connDetailString)
		if err != nil {
			return dbh, err
		}

		dbh.SetMaxIdleConns(50)
		dbh.SetMaxOpenConns(200)

		return
	}

	dbh, err = GetDatabaseHandleFromCfg(NewPostgresCfg(loginDetails))
	return
}

func GetDatabaseHandleFromCfg(cfg *PostgresCfg) (dbh *sql.DB, err error) {
	dbh, err = sql.Open("postgres", cfg.GetURI())
	if err != nil {
		return dbh, err
	}

	dbh.SetMaxIdleConns(50)
	dbh.SetMaxOpenConns(200)

	return dbh, err
}

func NewPostgresDataHandler(dbh *sql.DB,
	tableDetails map[string]interface{}) (pDH postgresDataHandler, err error) {
	if dbh == nil {
		return pDH, errors.New("Invalid database handle provided")
	}

	if tableDetails == nil {
		return pDH, errors.New("Invalid tableDetails provided")
	}

	if _, ok := tableDetails["table_name"]; !ok {
		return pDH, errors.New("table_name must be provided")
	}

	if _, ok := tableDetails["pk"]; !ok {
		return pDH, errors.New("Primary Key 'pk' details must be provided")
	}

	pDH = postgresDataHandler{}
	pDH.dbh = dbh
	pDH.tableName = tableDetails["table_name"].(string)
	pDH.projection = make(map[string]interface{})
	pDH.searchCriteria = make(map[string]interface{})

	pDH.batchSize = pDH.getDefaultBatchSize()
	pDH.EnforceBatch = true
	pDH.Record = make([]interface{}, 0)
	pDH.sortDirection = pDH.getDefaultSortDirection()

	for _, f := range tableDetails["pk"].([]string) {
		pDH.projection[f] = nil
		pDH.primaryKey = append(pDH.primaryKey, f)
	}

	return pDH, err
}

func (pDH *postgresDataHandler) getDefaultSortDirection() (rv string) {
	return "desc"
}

func (pDH *postgresDataHandler) getDefaultBatchSize() (rv int64) {
	return 10
}

func (pDH *postgresDataHandler) checkPrimaryKey() (err error) {
	for _, pkf := range pDH.primaryKey {
		found := false
		for scf, _ := range pDH.searchCriteria {
			if pkf == scf {
				found = true
				break
			}
		}

		if !found {
			return errors.New("All parts of the primary key must be provided")
		}
	}

	return err
}

func (pDH *postgresDataHandler) GetDMLStatement() (dmlstm string, dmlarg []interface{}) {
	return pDH.dmlStatement, pDH.dmlArguments
}

func (pDH *postgresDataHandler) SetBatchSize(b int64) {
	if b == 0 {
		pDH.batchSize = pDH.getDefaultBatchSize()
	} else {
		pDH.batchSize = b
	}
}

func (pDH *postgresDataHandler) SetProjection(fieldList []string) {
	for _, field := range fieldList {
		pDH.projection[field] = nil
	}
}

func (pDH *postgresDataHandler) SetFindCriteria(findKeys map[string]interface{}) (err error) {
	// Ensure a keyed search.  For now we'll restrict this to primary key specifically.
	for _, pkf := range pDH.primaryKey {
		if _, ok := findKeys[pkf]; !ok {
			return errors.New("Primary key field " + pkf + " is missing")
		}
	}

	pDH.searchCriteria = findKeys
	return err
}

func (pDH *postgresDataHandler) FindRecord(args ...string) (err error) {
	if pDH.dmlType != "" {
		return errors.New("Record is already staged for " + pDH.dmlType)
	}

	return_many := false
	for _, flag := range args {
		switch flag {
		case "return_many":
			return_many = true
		case "reverse_sort":
			if pDH.sortDirection == "desc" {
				pDH.sortDirection = "asc"
			} else {
				pDH.sortDirection = "desc"
			}
		}
	}

	if !return_many {
		if err := pDH.checkPrimaryKey(); err != nil {
			return err
		}
	}

	proj := []string{}
	for k, _ := range pDH.projection {
		proj = append(proj, k)
	}

	statement := bytes.NewBufferString("select ")

	// Add the projection
	statement.WriteString(strings.Join(proj, ","))

	statement.WriteString(" from " + pDH.tableName)

	// Process filtering crit
	if len(pDH.searchCriteria) > 0 {
		statement.WriteString(" where ")

		placeholders := []string{}

		pc := int64(1)
		for k, v := range pDH.searchCriteria {
			if reflect.TypeOf(v).Kind() == reflect.Slice {
				if len(v.([]interface{})) == 2 && stringIn((v.([]interface{}))[0].(string), []string{"<", "<=", "=>", ">"}) {
					fmt.Println(v)
				} else if len(v.([]interface{})) == 3 && stringIn((v.([]interface{}))[0].(string), []string{"between", "<betweeen", "between>", "<between>"}) {
					fmt.Println(v)
				}
			} else {
				placeholders = append(placeholders, k+"=$"+strconv.FormatInt(pc, 10))
				pDH.dmlArguments = append(pDH.dmlArguments, v)
				pc += 1
			}
		}

		statement.WriteString(strings.Join(placeholders, " and "))
	}

	// Process ordering crit
	statement.WriteString(" order by")
	for _, field := range pDH.primaryKey {
		statement.WriteString(" " + field + " " + pDH.sortDirection)
	}

	if pDH.EnforceBatch == true {
		statement.WriteString(" limit " + fmt.Sprint(pDH.batchSize+1))
	}

	pDH.dmlStatement = statement.String()

	rows, err := pDH.dbh.Query(pDH.dmlStatement, pDH.dmlArguments...)
	if err != nil {
		return err
	}

	var row map[string]interface{}
	columns, _ := rows.Columns()
	count := len(columns)

	counter := int64(0)
	for rows.Next() {
		values := make([]interface{}, count)
		valuePtrs := make([]interface{}, count)
		for i, _ := range columns {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)

		row = make(map[string]interface{})

		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}

			row[col] = v
		}
		counter += 1

		if counter <= pDH.batchSize || !pDH.EnforceBatch {
			pDH.Record = append(pDH.Record, row)
		}
	}

	pDH.NumAffectedLastOp = counter
	if counter > 0 {
		if counter > pDH.batchSize {
			pDH.RecordNextIdx = make(map[string]interface{})
			for _, field := range pDH.primaryKey {
				pDH.RecordNextIdx[field] = row[field]
			}
		}

		pDH.RecordLastOp = row
	}

	return err
}

func (pDH *postgresDataHandler) ExecuteProc(procName string,
	procArgs []interface{},
	args ...string) (err error) {

	statement := bytes.NewBufferString("select " + procName + "(")
	if procArgs != nil && len(procArgs) > 0 {
		placeholders := []string{}
		pc := int64(1)
		for _, v := range procArgs {
			placeholders = append(placeholders, "$"+strconv.FormatInt(pc, 10))
			pDH.dmlArguments = append(pDH.dmlArguments, v)
			pc += 1
		}
		statement.WriteString(strings.Join(placeholders, ","))
	}
	statement.WriteString(")")

	pDH.dmlStatement = statement.String()

	rows, err := pDH.dbh.Query(pDH.dmlStatement, pDH.dmlArguments...)
	if err != nil {
		return err
	}

	var row map[string]interface{}
	columns, _ := rows.Columns()
	count := len(columns)

	counter := int64(0)
	for rows.Next() {
		values := make([]interface{}, count)
		valuePtrs := make([]interface{}, count)
		for i, _ := range columns {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)

		row = make(map[string]interface{})

		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}

			row[col] = v
		}
		counter += 1

		pDH.Record = append(pDH.Record, row)
	}

	pDH.NumAffectedLastOp = counter
	if counter > 0 {
		if counter > pDH.batchSize {
			pDH.RecordNextIdx = make(map[string]interface{})
			for _, field := range pDH.primaryKey {
				pDH.RecordNextIdx[field] = row[field]
			}
		}

		pDH.RecordLastOp = row
	}

	return err
}

// Allow an update of a single record
func (pDH *postgresDataHandler) UpdateRecord(replacement map[string]interface{},
	args ...string) (err error) {
	if err := pDH.checkPrimaryKey(); err != nil {
		return err
	}

	if replacement == nil || len(replacement) == 0 {
		return errors.New("A replacement must be provided to update")
	}

	statement := bytes.NewBufferString("update " + pDH.tableName + " set\n    ")

	update_placeholders := []string{}
	pc := int64(1)

	for k, v := range replacement {
		if reflect.TypeOf(v).Kind() == reflect.Slice {
			// TODO handle things like increments ( field = field + 1 ) and the like
		} else {
			update_placeholders = append(update_placeholders, k+"=$"+strconv.FormatInt(pc, 10))
			pDH.dmlArguments = append(pDH.dmlArguments, v)
			pc += 1
		}
	}

	find_placeholders := []string{}
	for k, v := range pDH.searchCriteria {
		if reflect.TypeOf(v).Kind() == reflect.Slice {
			// TODO filtering crit
		} else {
			find_placeholders = append(find_placeholders, k+"=$"+strconv.FormatInt(pc, 10))
			pDH.dmlArguments = append(pDH.dmlArguments, v)
			pc += 1
		}
	}

	statement.WriteString(strings.Join(update_placeholders, ",\n    "))
	statement.WriteString("\nwhere\n    ")
	statement.WriteString(strings.Join(find_placeholders, "\nand "))

	pDH.dmlStatement = statement.String()

	res, err := pDH.dbh.Exec(pDH.dmlStatement, pDH.dmlArguments...)
	if err != nil {
		return err
	}

	if pDH.NumAffectedLastOp, err = res.RowsAffected(); err != nil {
		return err
	}

	return err
}

// Allow deletion of a single record
func (pDH *postgresDataHandler) DeleteRecord(args ...string) (err error) {
	// Allow only one to be deleted at the moment
	if err := pDH.checkPrimaryKey(); err != nil {
		return err
	}

	// We could have this return the deleted record
	statement := bytes.NewBufferString("delete from " + pDH.tableName + "\nwhere\n    ")
	placeholders := []string{}
	pc := int64(1)

	// This can contain additional filtering criteria beyond the primary key
	for k, v := range pDH.searchCriteria {
		if reflect.TypeOf(v).Kind() == reflect.Slice {
			// TODO
		} else {
			placeholders = append(placeholders, k+"=$"+strconv.FormatInt(pc, 10))
			pDH.dmlArguments = append(pDH.dmlArguments, v)
			pc += 1
		}
	}

	statement.WriteString(strings.Join(placeholders, "\nand "))

	pDH.dmlStatement = statement.String()
	fmt.Println(pDH.dmlStatement)

	res, err := pDH.dbh.Exec(pDH.dmlStatement, pDH.dmlArguments...)
	if err != nil {
		return err
	}

	if pDH.NumAffectedLastOp, err = res.RowsAffected(); err != nil {
		return err
	}

	return err
}

func (pDH *postgresDataHandler) InsertRecord(record map[string]interface{}, args ...string) (err error) {
	if pDH.dmlType != "" {
		return errors.New("Record is already stageed for " + pDH.dmlType)
	}

	if len(record) == 0 {
		return errors.New("A blank record was passed")
	}

	return_modified := false
	for _, flag := range args {
		if flag == "return_modified" {
			return_modified = true
		}
	}

	statement := bytes.NewBufferString("insert into " + pDH.tableName + " (")
	fields := []string{}
	placeholders := []string{}
	var values []interface{}

	counter := int64(1)
	for k, v := range record {
		fields = append(fields, k)
		values = append(values, v)
		placeholders = append(placeholders, "$"+strconv.FormatInt(counter, 10))
		counter += 1
	}
	statement.WriteString(strings.Join(fields, ","))
	statement.WriteString(") values (")
	statement.WriteString(strings.Join(placeholders, ","))
	statement.WriteString(")")

	pDH.dmlType = "insert"
	pDH.dmlArguments = values
	if return_modified {
		pDH.dmlStatement = fmt.Sprintf("with inserted_rows as (%s returning %s.*) select * from inserted_rows", statement.String(), pDH.tableName)
		//rowData := make(map[string]interface{})
		rows, err := pDH.dbh.Query(pDH.dmlStatement, pDH.dmlArguments...)
		if err != nil {
			return err
		}

		columns, _ := rows.Columns()
		count := len(columns)

		var row map[string]interface{}

		counter := int64(0)
		for rows.Next() {
			values := make([]interface{}, count)
			valuePtrs := make([]interface{}, count)
			for i, _ := range columns {
				valuePtrs[i] = &values[i]
			}
			rows.Scan(valuePtrs...)

			row = make(map[string]interface{})

			for i, col := range columns {
				var v interface{}
				val := values[i]
				b, ok := val.([]byte)
				if ok {
					v = string(b)
				} else {
					v = val
				}

				row[col] = v
			}
			counter += 1

			pDH.Record = append(pDH.Record, row)
		}

		pDH.NumAffectedLastOp = counter
		if counter > 0 {
			if counter > pDH.batchSize {
				pDH.RecordNextIdx = make(map[string]interface{})
				for _, field := range pDH.primaryKey {
					pDH.RecordNextIdx[field] = row[field]
				}
			}
			pDH.RecordLastOp = row
		}
	} else {
		pDH.dmlStatement = statement.String()
		_, err = pDH.dbh.Exec(pDH.dmlStatement, pDH.dmlArguments...)
	}

	return err
}
