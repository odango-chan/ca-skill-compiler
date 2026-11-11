package v01

import (
 "crypto/sha256"
 "encoding/base64"
 "encoding/binary"
 "encoding/json"
 "fmt"
 "os"
 "sort"
 "strings"
 "testing"
)

type vectorFile struct {
 Path string `json:"path"`
 Bytes string `json:"bytes"`
 Size int `json:"size"`
 SHA256 string `json:"sha256"`
}
type goldenVector struct {
 Name string `json:"name"`
 Files []vectorFile `json:"files"`
 SnapshotDigest string `json:"snapshotDigest"`
}
type goldens struct {
 Version string `json:"version"`
 Cases []goldenVector `json:"cases"`
}

// Test-only independent reference implementation for the target contract.
// It deliberately does not claim that the pre-v0.1 Compiler uses this hash.
func digestFiles(files []vectorFile) (string,error) {
 items:=append([]vectorFile(nil),files...)
 sort.Slice(items,func(i,j int)bool{return items[i].Path<items[j].Path})
 h:=sha256.New()
 _,_=h.Write([]byte("skillc/snapshot/v1\x00"))
 var buf [8]byte
 binary.BigEndian.PutUint32(buf[:4],uint32(len(items)))
 _,_=h.Write(buf[:4])
 previous:=""
 for i,f:=range items {
  if i>0 && f.Path==previous {return "",fmt.Errorf("duplicated path %q",f.Path)}
  previous=f.Path
  b,err:=base64.StdEncoding.DecodeString(f.Bytes)
  if err!=nil{return "",err}
  if len(b)!=f.Size{return "",fmt.Errorf("size mismatch for %s",f.Path)}
  sum:=sha256.Sum256(b)
  if got:="sha256:"+fmt.Sprintf("%x",sum);got!=f.SHA256{return "",fmt.Errorf("file hash mismatch for %s",f.Path)}
  path:=[]byte(f.Path)
  binary.BigEndian.PutUint32(buf[:4],uint32(len(path)))
  _,_=h.Write(buf[:4])
  _,_=h.Write(path)
  binary.BigEndian.PutUint64(buf[:],uint64(len(b)))
  _,_=h.Write(buf[:])
  _,_=h.Write(sum[:])
 }
 return "sha256:"+fmt.Sprintf("%x",h.Sum(nil)),nil
}

func TestSnapshotGoldenVectors(t *testing.T){
 raw,err:=os.ReadFile("snapshot-vectors.json")
 if err!=nil{t.Fatal(err)}
 var g goldens
 if err=json.Unmarshal(raw,&g);err!=nil{t.Fatal(err)}
 if g.Version!="skillc/snapshot/v1" || len(g.Cases)==0 {t.Fatalf("unexpected vectors header: %+v",g)}
 for _,tc:=range g.Cases{
  t.Run(tc.Name,func(t *testing.T){
   got,err:=digestFiles(tc.Files)
   if err!=nil{t.Fatal(err)}
   if got!=tc.SnapshotDigest{t.Fatalf("digest %s, want %s",got,tc.SnapshotDigest)}
   reversed:=append([]vectorFile(nil),tc.Files...)
   for i,j:=0,len(reversed)-1;i<j;i,j=i+1,j-1{reversed[i],reversed[j]=reversed[j],reversed[i]}
   if again,err:=digestFiles(reversed);err!=nil||again!=got{t.Fatalf("order-sensitive digest: %s %v",again,err)}
   tampered:=append([]vectorFile(nil),tc.Files...)
   tampered[0].Bytes=strings.TrimSuffix(tampered[0].Bytes,"==")+"AA=="
   if _,err:=digestFiles(tampered);err==nil{t.Fatal("tampered bytes must fail pinned file size/hash")}
  })
 }
}

func TestContractSchemasParse(t *testing.T){
 for _,file:=range []string{"project-settings.schema.json","tool-contract.schema.json","build-request.schema.json","resolver-result.schema.json","project-context.schema.json"}{
  raw,err:=os.ReadFile(file)
  if err!=nil{t.Fatal(err)}
  var v map[string]any
  if err=json.Unmarshal(raw,&v);err!=nil{t.Fatalf("%s: %v",file,err)}
  if v["$schema"]!="https://json-schema.org/draft/2020-12/schema"{t.Fatalf("%s missing Draft 2020-12",file)}
  if _,ok:=v["$defs"];!ok{t.Fatalf("%s missing defs",file)}
 }
}
