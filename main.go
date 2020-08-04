package main

import (
	"database/sql/driver"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"strings"
)

/*
CREATE TABLE `user`
(
    `id`   BIGINT(20) NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(20) DEFAULT '',
    `age`  INT(11)     DEFAULT '0',
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 1
  DEFAULT CHARSET = utf8mb4;
*/

var db *sqlx.DB

func initDB() (err error) {
	dsn := "root:root@tcp(127.0.0.1:3306)/study-sqlx?charset=utf8mb4&parseTime=True"
	// 也可以使用MustConnect连接不成功就panic
	db, err = sqlx.Connect("mysql", dsn)
	if err != nil {
		fmt.Printf("connect DB failed, err:%v\n", err)
		return
	}
	db.SetMaxOpenConns(200)
	db.SetMaxIdleConns(10)
	return
}

type user struct {
	ID   int    `db:"id"`
	Age  int    `db:"age"`
	Name string `db:"name"`
}

/*
查询
*/
// 查询单条数据示例
func queryRowDemo() {
	sqlStr := "select id, name, age from user where id=?"
	var u user
	err := db.Get(&u, sqlStr, 1)
	if err != nil {
		fmt.Printf("get failed, err:%v\n", err)
		return
	}
	fmt.Printf("id:%d name:%s age:%d\n", u.ID, u.Name, u.Age)
}

// 查询多条数据示例
func queryMultiRowDemo() {
	sqlStr := "select id, name, age from user where id > ?"
	var users []user
	err := db.Select(&users, sqlStr, 0)
	if err != nil {
		fmt.Printf("query failed, err:%v\n", err)
		return
	}
	fmt.Printf("users:%#v\n", users)
}

/*插入、更新和删除
sqlx中的exec方法与原生sql中的exec使用基本一致：
*/
// 插入数据
func insertRowDemo() {
	sqlStr := "insert into user(name, age) values (?,?)"
	ret, err := db.Exec(sqlStr, "沙河小王子", 19)
	if err != nil {
		fmt.Printf("insert failed, err:%v\n", err)
		return
	}
	theID, err := ret.LastInsertId() // 新插入数据的id
	if err != nil {
		fmt.Printf("get lastinsert ID failed, err:%v\n", err)
		return
	}
	fmt.Printf("insert success, the id is %d.\n", theID)
}

// 更新数据
func updateRowDemo() {
	sqlStr := "update user set age=? where id = ?"
	ret, err := db.Exec(sqlStr, 39, 6)
	if err != nil {
		fmt.Printf("update failed, err:%v\n", err)
		return
	}
	n, err := ret.RowsAffected() // 操作影响的行数
	if err != nil {
		fmt.Printf("get RowsAffected failed, err:%v\n", err)
		return
	}
	fmt.Printf("update success, affected rows:%d\n", n)
}

// 删除数据
func deleteRowDemo() {
	sqlStr := "delete from user where id = ?"
	ret, err := db.Exec(sqlStr, 6)
	if err != nil {
		fmt.Printf("delete failed, err:%v\n", err)
		return
	}
	n, err := ret.RowsAffected() // 操作影响的行数
	if err != nil {
		fmt.Printf("get RowsAffected failed, err:%v\n", err)
		return
	}
	fmt.Printf("delete success, affected rows:%d\n", n)
}

/*
NamedExec
DB.NamedExec方法用来绑定SQL语句与结构体或map中的同名字段。
*/
func insertUserDemo() (err error) {
	sqlStr := "INSERT INTO user (name,age) VALUES (:name,:age)"
	_, err = db.NamedExec(
		sqlStr,
		map[string]interface{}{
			"name": "小王",
			"age":  28,
		})
	return
}

/*
NamedQuery
与DB.NamedExec同理，这里是支持查询。
*/
func namedQuery() {
	sqlStr := "SELECT * FROM user WHERE name=:name"
	// 使用map做命名查询
	rows, err := db.NamedQuery(sqlStr, map[string]interface{}{"name": "小王"})
	if err != nil {
		fmt.Printf("db.NamedQuery failed, err:%v\n", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var u user
		err := rows.StructScan(&u)
		if err != nil {
			fmt.Printf("scan failed, err:%v\n", err)
			continue
		}
		fmt.Printf("user:%#v\n", u)
	}

	u := user{
		Name: "小王",
	}
	// 使用结构体命名查询，根据结构体字段的 db tag进行映射
	rows, err = db.NamedQuery(sqlStr, u)
	if err != nil {
		fmt.Printf("db.NamedQuery failed, err:%v\n", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var u user
		err := rows.StructScan(&u)
		if err != nil {
			fmt.Printf("scan failed, err:%v\n", err)
			continue
		}
		fmt.Printf("user:%#v\n", u)
	}
}

/*
事务操作
对于事务操作，我们可以使用sqlx中提供的db.Beginx()和tx.Exec()方法。示例代码如下：
*/
func transactionDemo2() (err error) {
	tx, err := db.Beginx() // 开启事务
	if err != nil {
		fmt.Printf("begin trans failed, err:%v\n", err)
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			fmt.Println("rollback")
			tx.Rollback() // err is non-nil; don't change it
		} else {
			err = tx.Commit() // err is nil; if Commit returns error update err
			fmt.Println("commit")
		}
	}()

	sqlStr1 := "Update user set age=20 where id=?"

	rs, err := tx.Exec(sqlStr1, 1)
	if err != nil {
		return err
	}
	n, err := rs.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("exec sqlStr1 failed")
	}
	sqlStr2 := "Update user set age=50 where id=?"
	rs, err = tx.Exec(sqlStr2, 50)
	if err != nil {
		return err
	}
	n, err = rs.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("exec sqlStr1 failed")
	}
	return err
}

//批量插入
/*
方法1：自己拼接语句实现批量插入（麻烦
比较笨，但是很好理解。就是有多少个User就拼接多少个(?, ?)。
*/
// BatchInsertUsers 自行构造批量插入的语句
func BatchInsertUsers(users []*user) error {
	// 存放 (?, ?) 的slice
	valueStrings := make([]string, 0, len(users))
	// 存放values的slice
	valueArgs := make([]interface{}, 0, len(users)*2)
	// 遍历users准备相关数据
	for _, u := range users {
		// 此处占位符要与插入值的个数对应
		valueStrings = append(valueStrings, "(?, ?)")
		valueArgs = append(valueArgs, u.Name)
		valueArgs = append(valueArgs, u.Age)
	}
	// 自行拼接要执行的具体语句
	stmt := fmt.Sprintf("INSERT INTO user (name, age) VALUES %s",
		strings.Join(valueStrings, ","))
	_, err := db.Exec(stmt, valueArgs...)
	return err
}

/*
方法2：使用sqlx.In实现批量插入（稍微简单，需实现接口
前提是需要我们的结构体实现driver.Valuer接口：
*/
func (u user) Value() (driver.Value, error) {
	return []interface{}{u.Name, u.Age}, nil
}

/*
使用sqlx.In实现批量插入代码如下：
*/
// BatchInsertUsers2 使用sqlx.In帮我们拼接语句和参数, 注意传入的参数是[]interface{}
func BatchInsertUsers2(users []interface{}) error {
	query, args, _ := sqlx.In(
		"INSERT INTO user (name, age) VALUES (?), (?), (?)",
		users..., // 如果arg实现了 driver.Valuer, sqlx.In 会通过调用 Value()来展开它
	)
	fmt.Println(query) // 查看生成的querystring
	fmt.Println(args)  // 查看生成的args
	_, err := db.Exec(query, args...)
	return err
}

// 方法3：BatchInsertUsers3 使用NamedExec实现批量插入（最简单， V1.2.1才发布该功能
func BatchInsertUsers3(users []*user) error {
	_, err := db.NamedExec("INSERT INTO user (name, age) VALUES (:name, :age)", users)
	return err
}

/*
sqlx.In的查询示例
关于sqlx.In这里再补充一个用法，在sqlx查询语句中实现In查询和FIND_IN_SET函数。即实现SELECT * FROM user WHERE id in (3, 2, 1);和SELECT * FROM user WHERE id in (3, 2, 1) ORDER BY FIND_IN_SET(id, '3,2,1');。

in查询
查询id在给定id集合中的数据。
*/
// QueryByIDs 根据给定ID查询
func QueryByIDs(ids []int) (users []user, err error) {
	// 动态填充id
	query, args, err := sqlx.In("SELECT name, age FROM user WHERE id IN (?)", ids)
	if err != nil {
		return
	}
	// sqlx.In 返回带 `?` bindvar的查询语句, 我们使用Rebind()重新绑定它
	query = db.Rebind(query)

	err = db.Select(&users, query, args...)
	return
}

/*
in查询和FIND_IN_SET函数
查询id在给定id集合的数据并维持给定id集合的顺序。
*/
// QueryAndOrderByIDs 按照指定id查询并维护顺序
func QueryAndOrderByIDs(ids []int) (users []user, err error) {
	// 动态填充id
	strIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		strIDs = append(strIDs, fmt.Sprintf("%d", id))
	}
	query, args, err := sqlx.In("SELECT name, age FROM user WHERE id IN (?) ORDER BY FIND_IN_SET(id, ?)", ids, strings.Join(strIDs, ","))
	if err != nil {
		return
	}

	// sqlx.In 返回带 `?` bindvar的查询语句, 我们使用Rebind()重新绑定它
	query = db.Rebind(query)

	err = db.Select(&users, query, args...)
	return
}

/*
当然，在这个例子里面你也可以先使用IN查询，然后通过代码按给定的ids对查询结果进行排序。
*/
func main() {
	if err := initDB(); err != nil {
		fmt.Printf("init DB failed, err:%v\n", err)
		return
	}
	fmt.Println("init DB success...")
	defer db.Close()
	//queryRowDemo()
	queryMultiRowDemo()
	//insertRowDemo()
	//updateRowDemo()
	//deleteRowDemo()
	//insertUserDemo()
	//namedQuery()
	//transactionDemo2()

	u1 := user{Name: "小王", Age: 18}
	u2 := user{Name: "xx", Age: 28}
	u3 := user{Name: "小王子", Age: 38}
	fmt.Println("----------------------------")
	// 方法1
	//users := []*user{&u1, &u2, &u3}
	//err := BatchInsertUsers(users)
	//if err != nil {
	//	fmt.Printf("BatchInsertUsers failed, err:%v\n", err)
	//}
	//fmt.Println("----------------------------")

	//// 方法2
	//users2 := []interface{}{u1, u2, u3}
	//err = BatchInsertUsers2(users2)
	//if err != nil {
	//	fmt.Printf("BatchInsertUsers2 failed, err:%v\n", err)
	//}
	//fmt.Println("----------------------------")

	// 方法3
	users3 := []*user{&u1, &u2, &u3}
	err := BatchInsertUsers3(users3)
	if err != nil {
		fmt.Printf("BatchInsertUsers3 failed, err:%v\n", err)
	}
	fmt.Println("----------------------------")

	//查询id在给定id集合中的数据。
	users4, err4 := QueryByIDs([]int{6, 5, 4})
	if err4 != nil {
		fmt.Printf("QueryByIDs failed, err:%v\n", err4)
	}
	for _, user := range users4 {
		fmt.Printf("user:%#v\n", user)
	}

	fmt.Println("----------------------------")

	//用mysql排序
	users5, err5 := QueryAndOrderByIDs([]int{6, 5, 4})
	if err4 != nil {
		fmt.Printf("QueryByIDs failed, err:%v\n", err5)
	}
	for _, user := range users5 {
		fmt.Printf("user:%#v\n", user)
	}
	fmt.Println("----------------------------")
	queryMultiRowDemo()
}
