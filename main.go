package main

import (
	"database/sql"
	_ "database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	nestedset "github.com/longbridgeapp/nested-set"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

type Document struct {
	ID            int64         `gorm:"PRIMARY_KEY;AUTO_INCREMENT" nestedset:"id"`
	ParentID      sql.NullInt64 `nestedset:"parent_id"`
	Rgt           int           `nestedset:"rgt"`
	Lft           int           `nestedset:"lft"`
	Depth         int           `nestedset:"depth"`
	ChildrenCount int           `nestedset:"children_count"`
	Title         string
}

const (
	getStatus      = "/api/status"
	createNewNode  = "/api/newnode"
	movingPath     = "/api/movefrom/{movefrom}/to/{moveto}"
	deleteNodePath = "/api/deletenode/{nodeID}"
)

var (
	db     Database
	host   = os.Getenv("POSTGRES_HOST")
	port   = os.Getenv("POSTGRES_PORT")
	user   = os.Getenv("POSTGRES_USER")
	pass   = os.Getenv("POSTGRES_PASSWORD")
	dbname = os.Getenv("POSTGRES_DB")
)

type Database struct {
	Connection *gorm.DB
}

func initDB() error { //TODO зачем возврашать ошибку?
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, dbname)
	dbCon, err := gorm.Open(postgres.Open(psqlInfo), &gorm.Config{})
	if err != nil {
		return err
	}
	fmt.Println("Соединение с БД устновлено")
	err = dbCon.AutoMigrate(&Document{})
	if err != nil {
		return err
	}
	fmt.Println("Таблица Document мигрирована")
	db.Connection = dbCon
	return nil
}

func ConnectionCheck(w http.ResponseWriter, r *http.Request) {
	sqlDB, err := db.Connection.DB()
	err = sqlDB.Ping()
	if err != nil {
		_, err := fmt.Fprint(w, "{\"Соединение установлено\": false}")
		if err != nil {
			return
		}
	} else {
		_, err := fmt.Fprint(w, "{\"Соединение установлено\": true}")
		if err != nil {
			return
		}
	}
}

func main() {
	err := initDB()
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	fmt.Printf("Host:%s\nPort:%s\nUser:%s\nPassword:%s\nDB:%s\n", host, port, user, pass, dbname)

	r.HandleFunc(getStatus, ConnectionCheck)
	r.HandleFunc(createNewNode, NewNodeWithParentsHandler)
	r.HandleFunc(movingPath, MoveHandler).Methods("GET")
	r.HandleFunc(deleteNodePath, deleteNode)

	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":80", nil))
}

func NewNodeWithParentsHandler(w http.ResponseWriter, r *http.Request) {
	var newDocument, doc Document

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения тела запроса", http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(requestBody, &newDocument)
	if err != nil {
		http.Error(w, fmt.Sprintf("Decode error: %v", err), http.StatusBadRequest)
		return
	}
	log.Printf("Request struct: %v\n", newDocument)

	if err = db.Connection.First(&doc, newDocument.ParentID.Int64).Error; err != nil {
		log.Printf("Документ с ключом %v не был найден\nОшибка: %v\n", newDocument.ParentID.Int64, err)
		w.WriteHeader(404)
		_, err = w.Write([]byte("Не существует документа с таким ID"))
		if err != nil {
			log.Printf("Не удалось передать запись в http\nОшибка: %v\n", err)
		}
	} else {
		if newDocument.ParentID.Valid {
			err = nestedset.Create(db.Connection, &newDocument, GetNodeByID(newDocument.ParentID.Int64))
			if err != nil {
				return
			}
			log.Println("Новый узел создан") //TODO change
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte("Новый узел создан"))
			if err != nil {
				log.Printf("Не удалось передать запись в http\nОшибка: %v\n", err)
			}
		} else {
			err = nestedset.Create(db.Connection, &newDocument, nil)
			if err != nil {
				return
			}
			log.Println("Создан новый нулевой узел") //TODO change
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte("Создан новый нулевой узел"))
			if err != nil {
				log.Printf("Не удалось передать запись в http\nОшибка: %v\n", err)
			}
		}
	}
	w.WriteHeader(http.StatusBadRequest)
}

func GetNodeByID(id int64) (x Document) {
	var doc Document
	db.Connection.Model(&Document{}).Where("id = ? ", id).First(&doc)
	return doc
}

func MoveHandler(w http.ResponseWriter, r *http.Request) { //TODO HANDLE ERRORS
	vars := mux.Vars(r)
	moveFrom, err := strconv.Atoi(vars["movefrom"])
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("moveFrom=%v\n", moveFrom)
	moveTo, err := strconv.Atoi(vars["moveto"])
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("moveTo=%v\n", moveTo)

	if err = nestedset.MoveTo(db.Connection, GetNodeByID(int64(moveFrom)), GetNodeByID(int64(moveTo)), nestedset.MoveDirectionInner); err != nil {
		log.Printf("Невозможно сделать перенос документа [%v], в документ [%v], ошибка:%v\n", moveFrom, moveTo, err)
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("Перенос невозможен"))
		if err != nil {
			log.Printf("Не удалось передать запись в http\nОшибка: %v\n", err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("Перенос выполнен успешно"))
		if err != nil {
			log.Printf("Не удалось передать запись в http\nОшибка: %v\n", err)
		}
	}
}

func deleteNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID, err := strconv.Atoi(vars["nodeID"])
	if err != nil {
		fmt.Println(err)
	}
	log.Printf("Trying to delele Node with ID =%v\n", nodeID)

	var node Document

	if err = db.Connection.First(&node, nodeID).Error; err != nil {
		log.Printf("Запись для удаление не была найдена\nОшибка: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("Запись для удаление не была найдена"))
		if err != nil {
			log.Printf("Не удалось передать запись в http\nОшибка: %v\n", err)
		}
	} else {
		if err = nestedset.Delete(db.Connection, &node); err != nil {
			log.Printf("Удаление узла не выполнено, ошибка: %v\n", err)
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte("Запись удалена"))
			if err != nil {
				log.Printf("Не удалось передать запись в http\nОшибка: %v\n", err)
			}
		}
	}
}

func getTables(w http.ResponseWriter, r *http.Request) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB"))
	db, err := gorm.Open(postgres.Open(psqlInfo), &gorm.Config{})
	if err != nil {
		_, err := fmt.Fprint(w, "{\"Соединение установлено\": false}")
		if err != nil {
			return
		}
	}
	sqlDB, err := db.DB()
	defer sqlDB.Close()

	var tables []struct {
		TableName string `gorm:"column:table_name"`
	}

	result := db.Raw("SELECT * FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tables)
	if result.Error != nil {
		_, err := fmt.Fprint(w, "{\"Соединение установлено\": false}")
		if err != nil {
			return
		}
	}
	response, err := json.Marshal(tables)
	if err != nil {
		_, err := fmt.Fprint(w, "{\"Соединение установлено\": false}")
		if err != nil {
			return
		}
	}

	_, err = fmt.Fprint(w, string(response))
	if err != nil {
		return
	}
}

//func getRowsFromTables(w http.ResponseWriter, r *http.Request) {
//	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
//		"password=%s dbname=%s sslmode=disable",
//		os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB"))
//	db, err := gorm.Open(postgres.Open(psqlInfo), &gorm.Config{})
//	if err != nil {
//		_, err := fmt.Fprint(w, "{\"Соединение установлено\": false}")
//		if err != nil {
//			return
//		}
//	}
//	// Получение всех записей из таблицы
//	var documents []Document
//	result := db.Find(&documents)
//	if result.Error != nil {
//		log.Fatal(result.Error)
//	}
//
//	// Вывод данных
//	for _, document := range documents {
//		fmt.Printf("Id: %d, Name: %s, ParantID: %d, Level: %d, Left_Key: %d, RightKey: %d\n", document.ID, document.Name, document.ParentId, document.Level, document.Leftkey, document.Rightkey)
//		result1 := []string{document.Name, strconv.FormatUint(uint64(document.ParentId), 10), strconv.FormatUint(uint64(document.Level), 10), strconv.FormatUint(uint64(document.Leftkey), 10), strconv.FormatUint(uint64(document.Rightkey), 10)}
//		response, _ := json.Marshal(result1)
//		_, err = fmt.Fprint(w, string(response))
//
//	}
//
//	// Закрытие соединения с базой данных
//	sqlDB, _ := db.DB()
//	defer sqlDB.Close()
//}

//func AddRowsInDocument(w http.ResponseWriter, r *http.Request) {
//	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
//		"password=%s dbname=%s sslmode=disable",
//		os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB"))
//	db, err := gorm.Open(postgres.Open(psqlInfo), &gorm.Config{})
//	if err != nil {
//		_, err := fmt.Println("{\"Соединение установлено\": false}")
//		if err != nil {
//			return
//		}
//	}
//	doc1 := Document{
//		Name:     "Document1",
//		ParentId: 0,
//		Level:    1,
//		Leftkey:  2,
//		Rightkey: 3,
//	}
//	db.Create(&doc1)
//	fmt.Fprint(w, "{\"Соединение установлено\": Добавлены строки в таблицу Documents}")
//
//	// Закрытие соединения с базой данных
//	sqlDB, _ := db.DB()
//	defer sqlDB.Close()
//}

func DeleteTableDocument(w http.ResponseWriter, r *http.Request) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB"))
	db, err := gorm.Open(postgres.Open(psqlInfo), &gorm.Config{})
	if err != nil {
		_, err := fmt.Println("{\"Соединение установлено\": false}")
		if err != nil {
			return
		}
	}
	// Создание таблицы
	err = db.Migrator().DropTable(&Document{})
	if err == nil {
		fmt.Fprint(w, "{\"Соединение установлено\": Таблица Documents удалена}")
	} else {
		fmt.Println("Failed to migrate table")
		return
	}

	// Закрытие соединения с базой данных
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
}
