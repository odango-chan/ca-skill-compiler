package compiler

// This file is the CA-SL v1alpha4 source/graph admission slice, not the
// v1alpha4 Compiler qualification. Owner + CapabilityContracts snapshot
// linking and §45.2 conformance are deliberately NOT implemented here.
//
// Frozen language: ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d.

import (
	"path"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const V1Alpha4Target = "connectionagent.io/v1alpha4"

var v4NodeID = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
var v4ContractGate = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*[.]preconditionpackage compiler

// This file is the CA-SL v1alpha4 source/graph admission slice, not the
// v1alpha4 Compiler qualification. Owner + CapabilityContracts snapshot
// linking and §45.2 conformance are deliberately NOT implemented here.
//
// Frozen language: ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d.

import (
	"path"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const V1Alpha4Target = "connectionagent.io/v1alpha4"

var v4NodeID = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
)

var v4Terminals = map[string]bool{"SUCCEEDED":true, "FAILED":true, "SAFE_STOPPED":true, "UNRESOLVED":true}
var v4Stages = map[string]bool{"precondition":true,"operation":true,"observation":true,"postcondition":true}
var v4ContractFields = map[string]bool{"goal":true,"precondition":true,"requestSchema":true,"contextSchema":true,"dependencies":true,"capabilities":true,"limits":true}
var v4PredicateFields = map[string]bool{"parameters":true,"evidence":true,"evaluator":true}

// CheckV1Alpha4 performs local source-structure checks and then fails closed
// when exact trusted Owner/CapabilityContracts inputs are unavailable.
// It is called only for an explicit BuildRequest.LanguageTarget.
func CheckV1Alpha4(req BuildRequest) []Diagnostic {
	return validateV1Alpha4(req).Diagnostics
}

func validateV1Alpha4(req BuildRequest) BuildResult {
	diagnostics:=make([]Diagnostic,0)
	errorAt:=func(file string,n *yaml.Node,code,message string){
		line,column:=0,0
		if n!=nil {line,column=n.Line,n.Column}
		diagnostics=append(diagnostics,Diagnostic{
			Severity:"error",Phase:"C_STATIC",Code:code,
			Path:limitDiagnosticPath(file),Line:line,Column:column,
			Message:limitDiagnosticMessage(message),
		})
	}
	if req.LanguageTarget!=V1Alpha4Target {
		errorAt("",nil,"LANGUAGE_TARGET_UNSUPPORTED","Unsupported CA-SL target: "+req.LanguageTarget)
		return BuildResult{Success:false,Diagnostics:diagnostics,OutputFiles:[]File{}}
	}
	files:=map[string][]byte{}
	skillNames:=map[string]bool{}
	for _,file:=range req.Files {
		if !validPath(file.Path) || path.Clean(file.Path)!=file.Path {
			errorAt(file.Path,nil,"PACKAGE_PATH_INVALID","Input must use normalized logical skills/<name> or declared dependency paths")
			continue
		}
		if _,ok:=files[file.Path];ok {
			errorAt(file.Path,nil,"YAML_PARSE_ERROR","Duplicate input file path")
			continue
		}
		if len(file.Bytes)>maxFileSize {
			errorAt(file.Path,nil,"YAML_PARSE_ERROR","File exceeds supported size")
			continue
		}
		files[file.Path]=file.Bytes
		parts:=strings.Split(file.Path,"/")
		if len(parts)>2 && parts[0]=="skills" {skillNames[parts[1]]=true}
	}
	if len(skillNames)==0 {
		errorAt("",nil,"SOURCE_ROOT_INVALID","No logical Skill package rooted at skills/<name>")
	}
	names:=make([]string,0,len(skillNames))
	for name:=range skillNames {names=append(names,name)}
	sort.Strings(names)
	for _,name:=range names {
		base:="skills/"+name+"/"
		contractPath:=base+"assurance/contract.yaml"
		procedurePath:=base+"assurance/procedure.yaml"
		predicatesPath:=base+"assurance/predicates.yaml"
		if files[base+"SKILL.md"]==nil {
			errorAt(base+"SKILL.md",nil,"SOURCE_ROOT_INVALID","SKILL.md missing")
		}
		contract:=v4ParseYAML(contractPath,files[contractPath],errorAt)
		procedure:=v4ParseYAML(procedurePath,files[procedurePath],errorAt)
		predicates:=v4ParseYAML(predicatesPath,files[predicatesPath],errorAt)
		if contract!=nil {
			v4Allowed(contract,contractPath,v4ContractFields,errorAt)
			if scalar(child(contract,"goal"))=="" {
				errorAt(contractPath,contract,"SOURCE_ROOT_INVALID","Contract requires goal")
			}
			if gate:=child(contract,"precondition");gate!=nil {
				if gate.Kind!=yaml.ScalarNode || gate.Tag!="!!str" || !v4ContractGate.MatchString(gate.Value) {
					errorAt(contractPath,gate,"ENTRY_PRECONDITION_REF_INVALID","precondition is one scalar node.precondition, not a list")
				}
			}
			if request:=child(contract,"requestSchema");request!=nil {
				ref:=scalar(request)
				if !strings.HasPrefix(ref,"schemas/") || files[base+ref]==nil {
					errorAt(contractPath,request,"REQUEST_SCHEMA_MISSING","Skill-owned request Schema is missing")
				}else{
					v4CheckSchema(base+ref,files[base+ref],errorAt)
				}
			}
			// A manifest alias string is never proof of the published Owner.
			if child(contract,"dependencies")!=nil {
				errorAt(contractPath,child(contract,"dependencies"),"DEPENDENCY_IDENTITY_UNVERIFIED",
					"Trusted DependencySnapshot Package ID/Owner/digest bindings are not yet supported by this v1alpha4 Compiler slice")
			}
			if child(contract,"capabilities")!=nil {
				errorAt(contractPath,child(contract,"capabilities"),"CAPABILITY_INPUT_UNTRUSTED",
					"Exact trusted v1alpha4 CapabilityContracts deployment inputs are not yet supported")
			}
		}
		if predicates!=nil {v4CheckPredicates(predicatesPath,predicates,errorAt)}
		if procedure!=nil {v4CheckProcedure(procedurePath,procedure,contract,errorAt)}
	}
	sort.Slice(diagnostics,func(i,j int)bool{
		a,b:=diagnostics[i],diagnostics[j]
		if a.Path!=b.Path{return a.Path<b.Path}
		if a.Line!=b.Line{return a.Line<b.Line}
		if a.Column!=b.Column{return a.Column<b.Column}
		return a.Code<b.Code
	})
	// A good source shape is NOT a successful Build. Do not emit artifacts
	// until the exact Owner/Schema/Evidence link closure is actually verified.
	if len(diagnostics)==0 {
		errorAt("",nil,"CAPABILITY_INPUT_UNTRUSTED",
			"v1alpha4 semantic/link/Owner qualification is unavailable in this Compiler slice")
	}
	return BuildResult{Success:false,Diagnostics:diagnostics,OutputFiles:[]File{}}
}

func v4ParseYAML(file string,data []byte,add func(string,*yaml.Node,string,string))*yaml.Node{
	if data==nil {
		add(file,nil,"SOURCE_ROOT_INVALID","Required v1alpha4 author source is missing")
		return nil
	}
	var document yaml.Node
	if err:=yaml.Unmarshal(data,&document);err!=nil {
		add(file,nil,"YAML_PARSE_ERROR",err.Error())
		return nil
	}
	if len(document.Content)!=1 || document.Content[0].Kind!=yaml.MappingNode {
		add(file,&document,"SOURCE_ROOT_INVALID","Expected YAML mapping source root")
		return nil
	}
	root:=document.Content[0]
	var checkDuplicate func(*yaml.Node)
	checkDuplicate=func(node *yaml.Node){
		if node==nil{return}
		if node.Kind==yaml.MappingNode {
			seen:=map[string]bool{}
			for i:=0;i+1<len(node.Content);i+=2 {
				key:=node.Content[i]
				if key.Kind!=yaml.ScalarNode || key.Tag!="!!str" || seen[key.Value] {
					add(file,key,"YAML_PARSE_ERROR","Invalid or duplicate YAML mapping key")
				}
				seen[key.Value]=true
			}
		}
		for _,item:=range node.Content {checkDuplicate(item)}
	}
	checkDuplicate(root)
	return root
}
func v4Allowed(node *yaml.Node,file string,allowed map[string]bool,add func(string,*yaml.Node,string,string)){
	if node==nil||node.Kind!=yaml.MappingNode {add(file,node,"SOURCE_ROOT_INVALID","Expected a mapping");return}
	for i:=0;i+1<len(node.Content);i+=2 {
		if !allowed[node.Content[i].Value] {
			add(file,node.Content[i],"UNKNOWN_FIELD","Unsupported v1alpha4 author field "+node.Content[i].Value)
		}
	}
}
func v4CheckSchema(file string,data []byte,add func(string,*yaml.Node,string,string)){
	root:=v4ParseYAML(file,data,add)
	if root==nil{return}
	if scalar(child(root,"$schema"))!="https://json-schema.org/draft/2020-12/schema" {
		add(file,root,"SCHEMA_INVALID","Schema dialect must be Draft 2020-12")
	}
	allowed:=map[string]bool{}
	for _,key:=range []string{"$schema","$id","title","description","default","examples","$ref","type","properties","required","additionalProperties","items","minItems","maxItems","minLength","maxLength","minimum","maximum","exclusiveMinimum","exclusiveMaximum","enum","const"}{allowed[key]=true}
	var walk func(*yaml.Node)
	walk=func(n *yaml.Node){
		if n==nil{return}
		if n.Kind==yaml.MappingNode {
			for i:=0;i+1<len(n.Content);i+=2{
				k,v:=n.Content[i],n.Content[i+1]
				// A property name is not a Schema keyword.
				if !allowed[k.Value] {
					add(file,k,"SCHEMA_INVALID","Unsupported G.4 keyword "+k.Value)
				}
				switch k.Value {
				case "properties":
					if v.Kind==yaml.MappingNode {
						for j:=1;j<len(v.Content);j+=2 {walk(v.Content[j])}
					}
				case "items":
					walk(v)
				}
			}
		}
	}
	walk(root)
}
func v4CheckPredicates(file string,root *yaml.Node,add func(string,*yaml.Node,string,string)){
	v4Allowed(root,file,map[string]bool{"predicates":true},add)
	set:=child(root,"predicates")
	if set==nil || set.Kind!=yaml.MappingNode {
		add(file,set,"SOURCE_ROOT_INVALID","PredicateSet must have predicates mapping")
		return
	}
	for i:=0;i+1<len(set.Content);i+=2 {
		def:=set.Content[i+1]
		v4Allowed(def,file,v4PredicateFields,add)
		params:=child(def,"parameters")
		if params==nil||params.Kind!=yaml.SequenceNode {
			add(file,params,"PREDICATE_GROUP_INVALID","parameters must be an ordered parameter list")
		}else for _,item:=range params.Content {
			if item.Kind!=yaml.MappingNode||len(item.Content)!=2 {
				add(file,item,"PREDICATE_GROUP_INVALID","Each parameter is a one-key declaration")
			}
		}
	}
}
func v4CheckProcedure(file string,root,contract *yaml.Node,add func(string,*yaml.Node,string,string)){
	v4Allowed(root,file,map[string]bool{"start":true,"nodes":true},add)
	start:=scalar(child(root,"start"))
	nodes:=child(root,"nodes")
	if nodes==nil||nodes.Kind!=yaml.MappingNode {
		add(file,nodes,"SOURCE_ROOT_INVALID","Procedure.nodes must be a mapping")
		return
	}
	ids:=map[string]bool{}
	for _,id:=range mapKeys(nodes) {ids[id]=true}
	if !ids[start] {add(file,child(root,"start"),"PROCEDURE_START_MISSING","start must resolve to a declared Node")}
	adjacency:=map[string][]string{}
	legalSuccess,anySuccess:=false,false
	for i:=0;i+1<len(nodes.Content);i+=2 {
		id,n:=nodes.Content[i].Value,nodes.Content[i+1]
		if !v4NodeID.MatchString(id) || v4Terminals[id] || id=="context" {
			add(file,nodes.Content[i],"NODE_ID_RESERVED","Invalid or reserved Node ID")
		}
		v4Allowed(n,file,v4Stages,add)
		stageOrder:=[]string{"precondition","operation","observation","postcondition"}
		nStage:=0
		for _,stage:=range stageOrder {if child(n,stage)!=nil{nStage++}}
		if nStage==0 {add(file,n,"SOURCE_ROOT_INVALID","Node must contain a stage")}
		for _,stage:=range stageOrder {
			body:=child(n,stage)
			if body==nil{continue}
			allowed:=map[string]bool{"routes":true}
			if stage=="precondition"||stage=="postcondition" {
				allowed["predicates"]=true
			}else{
				allowed["tool"]=true
				allowed["arguments"]=true
				if stage=="operation" {allowed["script"]=true;allowed["retryOf"]=true}
			}
			v4Allowed(body,file,allowed,add)
			if stage=="precondition"||stage=="postcondition" {
				calls:=child(body,"predicates")
				if calls==nil||calls.Kind!=yaml.SequenceNode||len(calls.Content)==0 {
					add(file,calls,"PREDICATE_GROUP_INVALID","Stage predicates must be a nonempty ordered list")
				}else for _,call:=range calls.Content {
					v4Allowed(call,file,map[string]bool{"predicate":true,"arguments":true},add)
					if scalar(child(call,"predicate"))=="" || child(call,"arguments")==nil || child(call,"arguments").Kind!=yaml.SequenceNode {
						add(file,call,"PREDICATE_GROUP_INVALID","PredicateCall requires predicate and positional arguments")
					}
				}
			}else{
				tool,script:=child(body,"tool"),child(body,"script")
				if (tool==nil)==(script==nil) || (stage=="observation"&&script!=nil) {
					add(file,body,"CONTROL_FLOW_AMBIGUOUS","Operation needs exactly one tool/script; Observation needs tool")
				}
				if args:=child(body,"arguments");args==nil || args.Kind!=yaml.MappingNode {
					add(file,body,"SOURCE_ROOT_INVALID","Tool or Script arguments must be a mapping")
				}else for j:=1;j<len(args.Content);j+=2 {
					v:=args.Content[j]
					if v.Kind!=yaml.ScalarNode {
						add(file,v,"SOURCE_PATH_INVALID","Arguments must be scalar SourcePaths or supported literals")
					}
				}
			}
			routes:=child(body,"routes")
			if routes!=nil&&routes.Kind!=yaml.MappingNode {
				add(file,routes,"SOURCE_ROOT_INVALID","routes must be a mapping")
				continue
			}
			if stage=="precondition"||stage=="postcondition" {
				for _,required:=range []string{"FAIL","UNKNOWN"} {
					if child(routes,required)==nil {
						add(file,body,"ROUTE_OUTCOME_MISSING","Predicate Stage missing "+required)
					}
				}
			}
			for _,outcome:=range mapKeys(routes) {
				target:=scalar(child(routes,outcome))
				if target=="" || (!ids[target]&&!v4Terminals[target]) {
					add(file,child(routes,outcome),"ROUTE_TARGET_INVALID","Unknown route target")
				}else if ids[target] {adjacency[id]=append(adjacency[id],target)}
				if target=="SUCCEEDED" {
					anySuccess=true
					if stage=="postcondition"&&outcome=="PASS" {legalSuccess=true}
				}
				if outcome=="OUTCOME_UNKNOWN"&&target!="UNRESOLVED"&&target!="SAFE_STOPPED" {
					add(file,child(routes,outcome),"ROUTE_OUTCOME_UNKNOWN","OUTCOME_UNKNOWN may only safely stop")
				}
			}
			// Owner Effect and SourcePath assignability remain unverified until exact
			// Tool/Schema publication. No guessed Owner result classification.
		}
	}
	if anySuccess&&!legalSuccess {
		add(file,root,"FINAL_POSTCONDITION_MISSING","Only a final Postcondition PASS may request SUCCEEDED")
	} else if legalSuccess {
		for _,id:=range mapKeys(nodes){
			n:=child(nodes,id)
			for _,stage:=range []string{"precondition","operation","observation","postcondition"}{
				for _,outcome:=range mapKeys(child(child(n,stage),"routes")){
					if scalar(child(child(child(n,stage),"routes"),outcome))=="SUCCEEDED" &&
						(stage!="postcondition"||outcome!="PASS") {
						add(file,child(child(n,stage),"routes"),"SUCCESS_BYPASSES_FINAL_POSTCONDITION","Non-final stage requests SUCCEEDED")
					}
				}
			}
		}
	}
	visited:=map[string]bool{}
	active:=map[string]bool{}
	cycle:=false
	var visit func(string)
	visit=func(id string){
		if active[id] {cycle=true;return}
		if visited[id] {return}
		visited[id]=true;active[id]=true
		for _,child:=range adjacency[id] {visit(child)}
		active[id]=false
	}
	if ids[start] {visit(start)}
	if cycle {add(file,root,"CONTROL_FLOW_CYCLE","Procedure must be a DAG")}
	if contract!=nil {
		gate:=scalar(child(contract,"precondition"))
		if gate!="" {
			parts:=strings.Split(gate,".")
			if len(parts)!=2 || !ids[parts[0]]||child(child(nodes,parts[0]),"precondition")==nil {
				add(file,child(contract,"precondition"),"ENTRY_PRECONDITION_REF_INVALID","Contract precondition must resolve to a declared Precondition stage")
			}
		}
	}
}
