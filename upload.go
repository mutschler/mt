package main

import (
    "bytes"
    "io"
    "io/ioutil"
    "mime/multipart"
    "net/http"
    "github.com/mutschler/mt/Godeps/_workspace/src/github.com/spf13/viper"
    log "github.com/mutschler/mt/Godeps/_workspace/src/github.com/sirupsen/logrus"
    "os"
)

// uploads a file via form submit to the given URL
func uploadFile(filename string) error {
  if viper.GetBool("upload") && viper.GetString("upload_url") != "http://example.com/upload" {
    log.Infof("uploading file...")
    targetUrl := viper.GetString("upload_url")
    bodyBuf := &bytes.Buffer{}
    bodyWriter := multipart.NewWriter(bodyBuf)

    // this step is very important
    fileWriter, err := bodyWriter.CreateFormFile("image", filename)
    if err != nil {
      log.Errorf("error writing to buffer")
      return err
    }

    // open file handle
    fh, err := os.Open(filename)
    if err != nil {
      log.Errorf("error opening file")
      return err
    }

    //iocopy
    _, err = io.Copy(fileWriter, fh)
    if err != nil {
      log.Errorf("error iocopy")
      return err
    }

    contentType := bodyWriter.FormDataContentType()
    bodyWriter.Close()

    resp, err := http.Post(targetUrl, contentType, bodyBuf)
    if err != nil {
      log.Errorf("error sending request to server")
      return err
    }
    defer resp.Body.Close()
    resp_body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
      log.Errorf("error reading server response")
      return err
    }
    log.Infof("Server response:\n%s", string(resp_body))
    return nil
  }
  return nil
}
