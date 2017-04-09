package main

import (
  // stuff for google authentication
  "golang.org/x/oauth2"
  "google.golang.org/api/drive/v3"
  "google.golang.org/api/googleapi"
  "golang.org/x/net/context"
  "golang.org/x/oauth2/google"
  
  "net/http"
  "io/ioutil"
  "fmt"
  "errors"
  "log"
  "encoding/json"
  "os"
  "strings"
  "html"
  "math/rand"
  "strconv"
)

const locJSON string = // file path to client_secret.json
const secretJSON string = // file path to api credentials json
const parentDirId string = // file id of parent directory.  Must be a folder. Must be owned by user.

var fs *drive.FilesService

func main() {
  // set up api
  driveService := getDriveService()
  fs = drive.NewFilesService(driveService)
  
  // set up server
  http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request){handleRequest(w,r)})
  log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleRequest(w http.ResponseWriter, r *http.Request){
  // parse the incoming string
  path := strings.TrimLeft(html.UnescapeString(r.URL.Path), "/")
  
  
  // get fildid from string
  file, err := getFileFromPath(path, parentDirId)
  
  if err != nil {
    fmt.Fprintf(w, "404 file not found")
    log.Println(path,"was not found")
    return
  }
  
  if isDirectoryFile(file) {
    //fmt.Fprintf(w, "%s\n", path)
    str := getDirContentsFromId(file.Id)
    log.Println("Directory",path,"has id",file.Id)
    fmt.Fprintf(w, "%s", str)
    return
  }
  
  if !isBinaryFile(file) {
    fmt.Fprintf(w, "404 file not found")
    log.Println(path,"not binary, had type",file.MimeType)
    return
  }

  // get contents of directory if it is a directory
  log.Println(path,"has id",file.Id)
  // get the contents of the file if it is a file
  str := getFileContentsFromId(file.Id)
  
  // if it is a google doc, return 404
  
  fmt.Fprintf(w, "%s", str);
}

func isBinaryFile(file *drive.File) bool {
  if file != nil {
    mime := file.MimeType
    return !strings.HasPrefix(mime, "application/vnd.google-apps")
  }
  return false
}

func isDirectoryFile(file *drive.File) bool {
  if file != nil {
    mime := file.MimeType
    return mime == "application/vnd.google-apps.folder"
  }
  return false
}

func getFileFromId(id string) (*drive.File, error) {
  // returns *FilesGetCall from file id
  getter := fs.Get(id)
  getter.Fields("id, name, parents, mimeType")
  
  // download the file
  random := strconv.Itoa(rand.Int())
  
  // this is terrible but works.  If the server gives an error, just try again
  var file *drive.File = nil
  var err error = errors.New("empty error")
  //for err != nil {
    file, err = getter.Do(googleapi.QuotaUser(random))
  //}
  handleError(err)
  return file, err
}

// used as a helper function
// finds the fileid from the given path and parent folder id
func getFileFromPath(path string, pid string) (*drive.File, error) {
  // check if this is just the parent directory
  if path == "" {
    log.Println("this is the parent directory")
    file, err := getFileFromId(parentDirId)
    return file, err
  }
  retChan := make(chan *drive.File)
  pidChan := make(chan string)
  go getFileFromPathConcur(path, pidChan, retChan)
  pidChan <- pid
  ret := <- retChan
  if ret == nil {
    err := errors.New("The file was not found.")
    return nil, err
  } else {
    return ret, nil
  }
}

// used to concurrently get the file from the path
func getFileFromPathConcur(path string, thisPidChan chan string, ret chan *drive.File) {
  // split into parts
  curName := strings.Split(path, "/")[0]
  rest := strings.TrimPrefix(path, curName)
  rest = strings.TrimPrefix(rest, "/")
  childPidChan := make(chan string)
  if rest != "" {
    // make recursive concurrent call
    go getFileFromPathConcur(rest, childPidChan, ret)
    // get list of files
  }
  files := listFilesWithName(curName)
  // get the pid of the file we are looking for
  thisPid := <- thisPidChan
  // find a file in our list that have this as the parent
  thisFile, err := searchFilesForPid(files, thisPid)
  // provide our child with our id
  if err != nil {
    // the file was not found
    if rest != "" {
      childPidChan <- ""
    } else {
      ret <- nil
    }
  } else {
    // the file was found
    if rest != "" {
      childPidChan <- thisFile.Id
    } else {
      ret <- thisFile
    }
  }
}

// returns array of files with the given name
func listFilesWithName(name string) []*drive.File {
  // set up the call
  list := fs.List()
  query := "name = '" + name + "'"
  list.Q(query)
  list.Fields("nextPageToken, files(id, name, parents, mimeType)")
  // make the call
  random := strconv.Itoa(rand.Int())
  
  // this is terrible but works.  If the server gives an error, just try again
  var files *drive.FileList = nil
  var err error = errors.New("empty error")
  for err != nil {
    files, err = list.Do(googleapi.QuotaUser(random))
  }
  // return the contained array
  return files.Files
}

func searchFilesForPid(files []*drive.File, pid string) (*drive.File, error){
  var ret *drive.File = nil
  for _, f := range files {
    for _, p := range f.Parents {
      if pid == p {
        ret = f
      }
    }
  }
  if ret != nil {
    return ret, nil
  } else {
    err := errors.New("No matching files found")
    return nil, err
  }
}
/*
func getContentsFromId(fs *drive.FileService, fileId string){
  
}
*/

// returns an html page with links to all the files in the folder
func getDirContentsFromId(dirId string) string {
  // make FilesListCall
  list := fs.List()
  // set query parameters
  query := "'"+dirId+"'"+" in parents"
  list.Q(query)
  // download the file
  random := strconv.Itoa(rand.Int())
  // this is terrible but works.  If the server gives an error, just try again
  var files *drive.FileList = nil
  var err error = errors.New("empty error")
  for err != nil {
    files, err = list.Do(googleapi.QuotaUser(random))
  }
  
  //files, err := list.Do()
  //handleError(err)
  // read it into a string
  
  // use this if you just want a string list of objects
  /*
  out := ""
  for _, f := range files.Files {
    out = out + f.Name + "\n"
  }
  */
  
  // use this for a formatted html page of contents
  out := "<head>\n</head>\n<body>\n"
  out = out + "<a href='../'>../</a></br>\n"
  for _, f := range files.Files {
    name := f.Name
    if isDirectoryFile(f) {name = name + "/"}
    out = out + "<a href=\"" + name + "\">" + name + "</a></br>\n"
  }
  out = out + "</body>"
  
  return out
}

// gets the file from the server and returns it as a []byte
// WARNING! this will spin forever if the fileId is not a file
// or anything else that would cause an error
// TODO: write to a stream rather than returning a byte object
func getFileContentsFromId(fileId string) []byte {
  // returns *FilesGetCall from file id
  getter := fs.Get(fileId)
  // download the file
  random := strconv.Itoa(rand.Int())
  
  // this is terrible but works.  If the server gives an error, just try again
  var resp *http.Response = nil
  var err error = errors.New("empty error")
  for err != nil {
    resp, err = getter.Download(googleapi.QuotaUser(random))
  }
  
  //handleError(err)
  // print the contents to the screen
  robots, err := ioutil.ReadAll(resp.Body)
  resp.Body.Close()
  if err != nil {
      log.Fatal(err)
  }
  return robots
}

func getDriveService() *drive.Service {
  // get secrets from json file
  f, err := os.Open(secretJSON)
  handleError(err)
  token := &oauth2.Token{}
  err = json.NewDecoder(f).Decode(token)
  defer f.Close()
  
  
  // get config from json
  cliSec, err := ioutil.ReadFile(locJSON)
  config, err := google.ConfigFromJSON(cliSec, "https://www.googleapis.com/auth/drive")
  handleError(err)
  
  
  // get client from config
  ctx := context.Background()
  client := config.Client(ctx, token)
  
  // get service from client
  driveService, err := drive.New(client)
  handleError(err)
  
  return driveService
}

func handleError(err error){
  if err != nil {
    log.Fatal(err.Error())
  }
}

