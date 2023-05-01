package main

import (
        "crypto/rand"
        "encoding/hex"

        "fmt"
        "log"
        "time"

        "github.com/jinzhu/gorm"
        _ "github.com/jinzhu/gorm/dialects/mysql"

        "net/http"
        "net/smtp"
        "github.com/gin-gonic/gin"

        "errors"
)

type Verify struct {
        Email           string  `gorm:"primary_key`
        Token           string  `gorm:"unique;not null"`
        Update_time     time.Time `gorm:"autoUpdateTime"`
}
type User struct {
        Email           string  `gorm:"primary_key`
        Password        string  `gorm:"not null"`
        Name            string  `gorm:"not null"`
        Gender          int     `gorm:"not null"`
        Age             int     `gorm:"not null"`
        Userid          string  `gorm:"unique;not null"`
        Teacher         string
}

type Teacher struct {
        Userid          string  `gorm:"primary_key"`
        Password        string  `gorm:"not null"`
}

var Database *gorm.DB

func initDatabase(){
        dsn := "root:12345678@tcp(172.16.20.10:3306)/cdt?charset=utf8mb4&parseTime=True&loc=Local"
        var err error
        Database, err = gorm.Open("mysql", dsn)
        if err != nil {
                log.Fatalf("failed to connect to database: %v", err)
        }
}

func generateToken() string {
        tokenBytes := make([]byte, 3)
        if _, err := rand.Read(tokenBytes); err != nil {
                log.Fatalf("failed to generate token: %v", err)
        }
        return hex.EncodeToString(tokenBytes)
}

func FindUserExistInTable(table interface{}, condition string, data ...interface{}) error{
        return Database.Where(condition, data...).First(table).Error
}

func DeleteUserExistInTable(table interface{},email string) error{
        return Database.Where("email = ?", email).Delete(table).Error
}

func AddUserInTable(record interface{}) error {
        return Database.Create(record).Error
}
func UpdateUserInTable(record interface{}, email string) error {
        //return Database.Where("email = ?", email).Updates(record).Error
        return Database.Table("verifies").Where("email = ?", email).Updates(record).Error
}

func SendVerify(email string, token string) error {
        from := "cdt.offic1@gmail.com"           // 你的 Gmail 地址
        password := "cwoafvfxipjzamjt"        // 你的 Gmail 密碼
        to := email      // 收件人電子郵件地址
        subject := "email verification\n" // 主題
        body := fmt.Sprintf("Please verify your email address \n your verification code %s", token)
        auth := smtp.PlainAuth("", from, password, "smtp.gmail.com")
        message := []byte(subject + "\n" + body)
        err := smtp.SendMail("smtp.gmail.com:587", auth, from, []string{to}, message)
        return err
}
/////////////////狀態選則器/////////////////////
func StatusSelection(c *gin.Context ,err error){
        switch err.Error() {
        case "Email already register":
                c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
        case "Email already exist":
                c.JSON(http.StatusNotAcceptable, gin.H{"error": err.Error()})
        case "Verification failed", "Email or Password Error", "Token has expired":
                c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        default:
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        }
        log.Println(err.Error())
}
/////////////////註冊////////////////////////////////
func Register(email string) error{
        errUser := FindUserExistInTable(&User{}, "email = ?",email)
        errVerify := FindUserExistInTable(&Verify{}, "email = ?",email)
        if errUser == nil {
                return errors.New("Email already exist")
        }
        if errVerify == nil {
                if err := Reverify(email); err != nil {
                        return err
                } else {
                        return errors.New("Email already register")
                }
        }
        token := generateToken();
        userVerify := &Verify{
                Email:       email,
                Token:       token,
                Update_time: time.Now(),
        }
        if err := AddUserInTable(userVerify); err !=nil{
                return err
        }
        if err := SendVerify(email, token); err != nil {
                return err
        }
        return nil
}

func RegisterHandler(c *gin.Context) {
        // Parse the user's registration data from the request body.
        var userData struct {
                Email    string `json:"email" binding:"required,email"`
        }
        if err := c.ShouldBindJSON(&userData); err != nil {
                //log.Print("JSON ERROR\n")
                StatusSelection(c, err)
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        if err := Register(userData.Email); err != nil {
                //log.Print("REGISTER ERROR\n")
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        c.JSON(http.StatusOK, gin.H{"message": "registration successful"})
}

//////////////////////////驗證///////////////////////////////
func Validator(email string, token string) error{
        var userVerify =  &Verify{}
        if err := FindUserExistInTable(userVerify,"email = ?",email); err != nil{
                return err
        }
        if time.Now().After(userVerify.Update_time.Add(5 * time.Minute)) {
                return errors.New("Token has expired")
        }
        if(userVerify.Email==email && userVerify.Token==token){
                if err := DeleteUserExistInTable(&Verify{},email); err != nil{
                        return err
                }
        } else {
                return errors.New("Verification failed")
        }
        return nil

}
func Enroll(user *User) error{
        return AddUserInTable(user)
}
func EnrollHandler(c *gin.Context) {
        var userData struct {
                Email    string `json:"email" binding:"required,email"`
                Password string `json:"password" binding:"required,min=8"`
                Name     string `json:"name" binding:"required,max=15,regexp=^[\p{Han}]+$"`
                Gender   int    `json:"gender" binding:"required,oneof=1 2 3"`
                Age      int    `json:"age" binding:"required,min=18"`
                Token    string `json:"token" binding:"required,max=10"`
        }
        if err := c.ShouldBindJSON(&userData); err != nil {
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        if err := Validator(userData.Email, userData.Token); err!= nil {
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        userid := generateToken();
        var userTeacher = &Teacher{}
        if err := Database.Order("RAND()").First(userTeacher).Error; err != nil {
                StatusSelection(c, err)
                return
        }
        userUser := &User{Email: userData.Email, Password: userData.Password, Name: userData.Name, Gender: userData.Gender, Age: userData.Age, Userid: userid, Teacher: userTeacher.Userid}
        if err := Enroll(userUser); err!= nil {
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        c.JSON(http.StatusOK, gin.H{"message": "successfully"})
}
/////////////////////////////重新發送驗證碼///////////////////////////
func Reverify(email string) error {
        var userVerify =  &Verify{}
        if err := FindUserExistInTable(userVerify,"email = ?",email); err != nil{
                return err
        }
        token := generateToken();
        userVerify.Token = token
        userVerify.Update_time = time.Now()
        if err := SendVerify(email, token); err != nil {
                return err
        }
        if err := UpdateUserInTable(userVerify,email); err !=nil{
                log.Print("ADD ERROR\n")
                return err
        }
        return nil
}
func ReverifyHandler(c *gin.Context) {
        // Parse the user's registration data from the request body.
        var userData struct {
                Email    string `json:"email" binding:"required,email"`
        }
        if err := c.ShouldBindJSON(&userData); err != nil {
                //log.Print("JSON ERROR\n")
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        if err := Reverify(userData.Email); err != nil {
                //log.Print("REGISTER ERROR\n")
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        c.JSON(http.StatusOK, gin.H{"message": "Verification code resent successfully"})
}
///////////////////登錄/////////////////////////////
func Login(email string, password string) (error, string, string){
        var userUser =  &User{}
        if err := FindUserExistInTable(userUser,"email = ?",email); err != nil{
                return err, "", ""
        }
        if !(userUser.Email==email && userUser.Password==password){
                return errors.New("Email or Password Error"), "" ,""
        }
        return nil,userUser.Userid,userUser.Teacher
}

func LoginHandler(c *gin.Context) {
        var userData struct {
                Email    string `json:"email" binding:"required,email"`
                Password string `json:"password" binding:"required,min=8"`
        }
        if err := c.ShouldBindJSON(&userData); err != nil {
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        err, uerid, teacher := Login(userData.Email, userData.Password)
        if err!= nil {
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        c.JSON(http.StatusOK, gin.H{"userid" : uerid, "teacher" : teacher})
}
/////////////////////////////GetTeacher////////////////////////////////////
/*
func SetTeacher(user interface{}) error{
        var userTeacher = &Teacher{}
        if err := Database.Order("RAND()").First(userTeacher).Error; err != nil {
                return err
        }
        userUser := user.(*User)
        userUser.Teacher = userTeacher.Userid
        if err := Database.Table("users").Where("email = ?", userUser.Email).Updates(userUser).Error; err != nil {
                log.Println(userUser.Email)
                return err
        }
        return nil
}
*/
func GetTeacherHandler(c *gin.Context) {
        var userData struct {
                Email    string `json:"email" binding:"required,email"`
                Password string `json:"password" binding:"required,min=8"`
        }
        if err := c.ShouldBindJSON(&userData); err != nil {
                StatusSelection(c, err)
                return
        }
        var userUser = &User{}
        if err := FindUserExistInTable(userUser,"email = ? AND  password = ?",userData.Email,userData.Password); err != nil{
                StatusSelection(c, errors.New("Email or Password Error"))
                return
        }
        var userTeacher = &Teacher{}
        if err := Database.Order("RAND()").First(userTeacher).Error; err != nil {
                StatusSelection(c, err)
                return
        }
        userUser.Teacher = userTeacher.Userid
        if err := Database.Table("users").Where("email = ?", userUser.Email).Updates(userUser).Error; err != nil {
                StatusSelection(c, err)
                return
        }
        c.JSON(http.StatusOK, gin.H{ "teacher" : userUser.Teacher})
}

/*
func GetTeacher(c *gin.Context, email string)(error, string){
        var randomTeacher Teacher
        if err := FindUserExistInTable(userUser,"email = ? AND  password = ?",userData.Email); err != nil{
                StatusSelection(c, errors.New("Email or Password Error"))
                return
        }
        return nil, randomTeacher.userid
}
*/
///////////////////////////UpdatePassword///////////////////////////

func UpdatePasswordHandler(c *gin.Context) {
        var userData struct {
                Email    string `json:"email" binding:"required,email"`
                Password string `json:"password" binding:"required,min=8"`
                NewPassword string `json:"newpassword" binding:"required,min=8"`
        }
        if err := c.ShouldBindJSON(&userData); err != nil {
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        var userUser =  &User{}
        if err := FindUserExistInTable(userUser,"email = ? AND  password = ?",userData.Email,userData.Password); err != nil{
                StatusSelection(c, errors.New("Email or Password Error"))
                return
        }
        userUser.Password = userData.NewPassword
        if err := Database.Table("users").Where("email = ?", userUser.Email).Updates(userUser).Error; err != nil {
                StatusSelection(c, err)
                return
        }
        c.JSON(http.StatusOK, gin.H{"message": "successful"})
}
/////////////////////////////UpdateName////////////////////////////
func UpdateNameHandler(c *gin.Context) {
        var userData struct {
                Email    string `json:"email" binding:"required,email"`
                Newname  string `json:"name" binding:"required,max=15,regexp=^[\p{Han}]+$"`
        }
        if err := c.ShouldBindJSON(&userData); err != nil {
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        var userUser =  &User{}
        if err := FindUserExistInTable(userUser,"email = ?",userData.Email); err != nil{
                StatusSelection(c, errors.New("Email or Password Error"))
                return
        }
        userUser.Name = userData.Newname
        if err := Database.Table("users").Where("email = ?", userUser.Email).Updates(userUser).Error; err != nil {
                StatusSelection(c, err)
                return
        }
        c.JSON(http.StatusOK, gin.H{"message": "successful"})
}
//////////////////////////////DeleteMail/////////////////////////
func DeleteHandler(c *gin.Context) {
        var userData struct {
                Email    string `json:"email" binding:"required,email"`
        }
        if err := c.ShouldBindJSON(&userData); err != nil {
                //log.Print("JSON ERROR\n")
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        if err := DeleteUserExistInTable(&User{},userData.Email); err != nil{
                StatusSelection(c, err)
        }
        c.JSON(http.StatusOK, gin.H{"message": "successful"})
}

func RootHandler(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "successful"})
}
/////////////////////teacherlogin/////////////////////////////
func TeacherLogin(userid string, password string) error {
        var teacherUser =  &Teacher{}
        if err := FindUserExistInTable(teacherUser,"userid = ?",userid); err != nil{
                return err
        }
        if !(teacherUser.Userid==userid && teacherUser.Password==password){
                return errors.New("User or Password Error")
        }
        return nil
}

func TeacherLoginHandler(c *gin.Context) {
        var teacherData struct {
                Userid   string `json:"userid"`
                Password string `json:"password" binding:"required,min=8"`
        }
        if err := c.ShouldBindJSON(&teacherData); err != nil {
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        err := TeacherLogin(teacherData.Userid, teacherData.Password)
        if err!= nil {
                //c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                StatusSelection(c, err)
                return
        }
        c.JSON(http.StatusOK,gin.H{})
}

func main() {
        // Initialize the database.
        initDatabase()
        defer Database.Close()
        // Create the Gin router and set up the API endpoints.
        router := gin.Default()
        router.POST("/register", RegisterHandler)
        router.POST("/enroll", EnrollHandler)
        router.POST("/login", LoginHandler)
        router.POST("/reverify", ReverifyHandler)
        router.POST("/getteacher", GetTeacherHandler)
        router.POST("/deletemail", DeleteHandler)
        router.POST("/updatepassword", UpdatePasswordHandler)
        router.POST("/updatename", UpdateNameHandler)
	router.POST("/teacherlogin", TeacherLoginHandler)
        router.GET("/", RootHandler)
        // Start the server.
        if err := router.Run(":80"); err != nil {
                log.Fatalf("failed to start server: %v", err)
        }
}

