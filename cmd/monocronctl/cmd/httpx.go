package cmd

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

func getJSON[T any](url string, out *T) error {
    resp, err := http.Get(url)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("%s: %s", resp.Status, string(b))
    }
    return json.NewDecoder(resp.Body).Decode(out)
}

func postJSON[Req any, Res any](url string, in *Req, out *Res) error {
    body, _ := json.Marshal(in)
    resp, err := http.Post(url, "application/json", bytes.NewReader(body))
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("%s: %s", resp.Status, string(b))
    }
    if out != nil { return json.NewDecoder(resp.Body).Decode(out) }
    return nil
}

func del(url string) error {
    req, _ := http.NewRequest(http.MethodDelete, url, nil)
    resp, err := http.DefaultClient.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 { b, _ := io.ReadAll(resp.Body); return fmt.Errorf("%s: %s", resp.Status, string(b)) }
    return nil
}

