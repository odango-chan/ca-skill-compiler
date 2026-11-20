# Development Build 鏈€缁堟灦鏋勪竴鑷存€ф敹鏁涳細瀹炵幇涓庨獙鏀舵姤鍛?
**鐘舵€侊細Draft PR #51 璇勫鐗堛€?* 鏈〉璁板綍 `feat/compiler-development-build-20261120` 涓婂疄鐜颁笌 [PR #48 鏋舵瀯鍩虹嚎](https://github.com/odango-chan/ca-skill-compiler/pull/48) 鐨勪竴鑷存€ц瘉鎹€傚畠涓嶅喕缁撳叕寮€濂戠害銆佷笉鍚堝苟銆佷笉鍙戝竷銆?
| 椤?| 鍊?|
|---|---|
| 璁捐濂戠害鎻愪氦锛圥R #48锛?| `551a0bba3455cfe4e7dbeede274b686a523316d6` |
| 鏈姤鍛婄敓鎴愭椂 PR #48 澶撮儴 | `2316b4e438dc693db80ae870f12363092dbe8f58` |
| 鏉冨▉鑱岃矗鍩虹嚎 | [`compiler-evidence-assurance-responsibility-audit.zh-CN.md`](compiler-evidence-assurance-responsibility-audit.zh-CN.md) |
| 浜у搧鐗堟湰 / 璇█鐩爣 | CA Skill Compiler v2.0.0锛堟湭鏀癸級/ `connectionagent.io/v1alpha5`锛堟湭鏀癸級 |
| 鍐荤粨鍏ュ彛 | v2.0.2 BuildRequest / BuildResult / Artifact 閫愬瓧鑺傛湭鏀?|
| MAS G0 / G1 | **NOT_RUN** |

## 1. 鏈疆瀹炵幇鑼冨洿

浠?`ca-skill-compiler`銆傛湭淇敼 ca-skills 鍐荤粨瑙勮寖銆丆ompiler v2.0.2 宸插彂甯冨叆鍙?Artifact銆乧a-agent銆乧a-agent-demo銆乧a-client-tools銆乪nv-openamsm锛屼篃鏈敼浠讳綍绋冲畾 ID 鎴栧巻鍙蹭骇鐗┿€?
### 1.1 Requirement 涓ょ被鍒嗙

- 鏂板鐙珛浜х墿 `runtime-execution-requirements.json`锛沗host-admission-requirements.json` 淇濈暀鍘熸湁 Host Admission 璇箟銆?- 绫诲埆鐢?`kind` 鍞竴纭畾锛?  - Host Admission锛坄HOST_VERIFICATION_REQUIRED`锛夛細`OWNER_TRUST`銆乣EVIDENCE_DOMAIN_TRUST`銆乣EVIDENCE_FACT_TRUST`銆乣EVIDENCE_INDEPENDENCE_TRUST`銆?  - Runtime Execution锛坄RUNTIME_EXECUTION_REQUIRED`锛夛細`COMMIT_PROOF`銆乣RECONCILE`銆?- `COMMIT_PROOF` / `RECONCILE` 鐢卞悓涓€涓?collector 鑷姩璺敱鍒?Runtime 鏂囦欢锛屼笉鍐嶈繘鍏?Host Admission 闆嗗悎銆?- `manifest.json` 鏂板 `requirements { hostAdmission, runtimeExecution }`锛屾樉寮忓０鏄庝袱涓枃浠剁殑濂戠害韬唤銆?- `VerifyDevelopment` 鍚屾椂鏍￠獙涓や釜鏂囦欢锛欽CS 瑙勮寖鍖栥€佹帓搴忋€侀€愭潯閲嶇畻 `requirementId`銆乲ind/file/status 涓夎€呬竴鑷淬€乮ndependence 鏄庣粏缁撴瀯銆佽鑼冨寲绋嬪害锛涢殢鍚庝互鍘熷璇锋眰閲嶅缓骞堕€愬瓧鑺傛瘮杈冨叏閮ㄤ骇鐗┿€?- 瀹為檯 Commit Proof 涓庡疄闄?Reconcile 缁撹涓嶈繘鍏ョ紪璇戜骇鐗╋紝Compiler 涓嶈緭鍑轰换浣曗€滃凡鎻愪氦/宸叉仮澶嶁€濈殑鍒ゅ畾銆?
### 1.2 Evidence 澶氶噸 Requirement

- 灏嗗師鍏?`alias::exportID -> reason` 鐨勫崟鍊兼槧灏勬敼涓?`alias::exportID -> set(reason)`銆?- 鍚屼竴 `EvidenceDescriptor` 鍚屾椂澹版槑 `sourceIdentity` 涓?`factIdentity` 鏃讹紝鍒嗗埆浜х敓 `EVIDENCE_DOMAIN_TRUST` 涓?`EVIDENCE_FACT_TRUST`锛屼簰涓嶈鐩栥€?- 杈撳叆鏁扮粍閲嶆帓涓嶆敼鍙樹骇鐗╁瓧鑺傦紙纭畾鎬ф祴璇曡鐩栵級銆?
### 1.3 Independence 绮剧‘寤烘ā

- 姣忎釜 Independence Group 璁板綍涓€鏉?`EVIDENCE_INDEPENDENCE_TRUST`锛屾惡甯︼細

  ```
  independence { predicate, groupIndex, members[{dependencyAlias,key,exportId,sourcePath,domain}], domainPairs[[a,b],...] }
  ```

- 缁勪綅缃簿纭负 `/predicates/<predicate>/evidence/independence/<index>`锛涗笉鍐嶄娇鐢?`env.roots[0]` 鎴栧崟涓€甯冨皵鍊笺€?- 姣忎釜鎴愬憳瑙ｆ瀽鍒板敮涓€ Descriptor銆佸敮涓€ Interface Package key 鍜屽０鏄庡煙锛涘煙瀵归€愬鍒楀嚭骞惰鑼冨寲鎺掑簭銆?- 缁勬垚鍛樼己澶便€丏escriptor 涓嶅敮涓€銆佺己灏?`sourceIdentity`銆佸皯浜庝袱涓０鏄庡煙绛夐潤鎬侀潪娉曟儏褰㈢洿鎺?Build FAIL锛涘彧鏈夊閮?IndependentDomains 鎺堟潈缂哄け鎵嶈褰?Host Requirement銆?- 澶?Group銆佸 Root锛堜竴涓粍璺ㄤ袱涓牴鍖咃級銆佸 Domain Pair 鍧囨湁缁勫悎娴嬭瘯銆?
### 1.4 IngressCatalog 绮剧‘韬唤

`package.json` 鐨?`IngressCatalog` 瀵煎嚭鐜板湪蹇呴』涓庤浇鑽风簿纭粦瀹氾細

1. `apiVersion` / `kind` 绮剧‘锛?2. 瀵煎嚭 `id` 鎭板ソ瀵瑰簲 `spec.operations[].id` 涓殑涓€椤癸紱
3. 瀵煎嚭 `path` 灏辨槸瀹為檯 catalog 鏂囦欢锛?4. Operation ID 鍦ㄦ枃浠跺唴鍞竴锛?5. `metadata.owner` 绛変簬鍖?`publisher`锛?6. 鏂囦欢鍐呮瘡涓?operation 蹇呴』鍦ㄥ悓涓€璺緞鏈夌簿纭鍑猴紙鏃犳湭瀵煎嚭 operation銆佹棤 id/path 婕傜Щ锛夛紱
7. `artifactBindings` / `schemaBindings` 鐨勪緷璧栧埆鍚嶃€佸揩鐓ц矾寰勩€乤rtifactRef 涓?Schema 韬唤蹇呴』钀藉湪閿佸畾闂寘鍐呫€?
鍙︿慨澶?`legacyRequest` 鎶婃湭澹版槑渚濊禆榛橀粯鍥為€€涓?`self` 鐨勯殣寮忚矾寰勶紱鐜板湪鏈０鏄庣殑渚濊禆鍒悕鍦ㄥ噯鍏ラ樁娈靛嵆澶辫触銆?
## 2. 鏋舵瀯涓庡疄鐜板樊寮傛竻鍗?
| 鏋舵瀯濂戠害锛圥R #48锛?| 瀹炵幇锛圥R #51锛?| 宸紓 |
|---|---|---|
| 杈撳叆鍙湁婧愮爜銆佺簿纭?Interface Package 蹇収銆丏ependency Lock锛涙棤 `assurance[]` / 鍙€?Assurance Export | `DevelopmentBuildRequest` 鍙湁 `files/lock/snapshots`锛沗DevelopmentAssurance` 绫诲瀷宸插垹闄?| 鏃?|
| `EvidenceDescriptor` 鏄寮忓鍑猴紝缂哄け/閲嶅/闈炴硶/Producer 涓嶅尮閰嶅嵆 FAIL | `EvidenceDescriptor` 瀵煎嚭绱㈠紩 + 闈欐€侀摼鎺ワ紱澶辫触鍗虫棤浜х墿 | 鏃?|
| Host Admission 鍥涚被 kind + `HOST_VERIFICATION_REQUIRED` | `host-admission-requirements.json` 浠呮帴鍙楀洓绫?| 鏃?|
| Runtime 涓ょ被 kind + `RUNTIME_EXECUTION_REQUIRED` | `runtime-execution-requirements.json` 浠呮帴鍙椾袱绫?| 鏃?|
| Requirement 璁板綍褰㈢姸鍚?location/subjects/status锛宨ndependence 鍙﹀甫鏄庣粏 | 瀛楁銆佹帓搴忋€佹憳瑕佸煙涓庢枃妗ｄ竴鑷?| 鏃?|
| 涓€涓?Independence Group 涓€鏉?Requirement锛岀簿纭?selector | 瀹炵幇鎸夌粍鐢熸垚锛宻elector 绮剧‘鍒扮粍搴忓彿 | 鏃?|
| 澶氫俊浠讳簨瀹炰笉浜掔浉瑕嗙洊 | reason 闆嗗悎瀹炵幇锛屼袱鏉″苟瀛?| 鏃?|
| Ingress 瀵煎嚭韬唤涓冩潯 | 涓冩潯鍏ㄩ儴鍦?`validateDevelopmentPackageExports` / `validateDevelopmentBindingTargets` 瀹炴柦 | 鏃?|
| Artifact 鎭?`NOT_ADMITTED`锛岀敓浜?Host 鎷掓敹 `DEVELOPMENT` | 鏈敼锛涙祴璇曞浐瀹氭墽琛岀姸鎬?| 鏃?|
| Verify 鍚屾椂妫€鏌ヤ袱绫诲苟閫愬瓧鑺傞噸寤?| 瀹炵幇 | 鏃?|

鏈彂鐜版灦鏋勬枃妗ｄ笌瀹炵幇涔嬮棿鐨勬湭鍐冲樊寮傘€傚凡鐭ョ殑闈炲绾﹀樊寮傚彧鏈夋祴璇曞す鍏风殑缁勭粐鏂瑰紡锛坄developmentRequestFromLegacy` 涓烘祴璇曚笓鐢ㄨ浆鎹紝涓嶆瀯鎴愬澶栨帴鍙ｏ級銆?
## 3. 娴嬭瘯鎶ュ憡锛堝疄闄呮墽琛岋級

鐜锛歐indows锛汫o **go1.26.6 windows/amd64**锛汵ode **v24.18.0**锛汫oogle Chrome **158.0.8127.37**锛汚rchify 娓叉煋鍣?**3.2.0**銆?
| 闂ㄧ | 鍛戒护 | 缁撴灉 |
|---|---|---|
| 鍗曞厓/闆嗘垚 | `GOWORK=off go test ./... -count=1` | PASS锛堝叏閮ㄥ寘锛?|
| 闈欐€佹鏌?| `GOWORK=off go vet ./...` | PASS |
| Native build | `CGO_ENABLED=0 GOWORK=off go build ./...` | PASS |
| Go/WASM build | `CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm` | PASS |
| CLI E2E | `GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v` | PASS |
| Chromium Worker parity | `CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v` | PASS锛圖evelopment 5 鏂囦欢閫愬瓧鑺備竴鑷达級 |
| 绌虹櫧/缂栫爜 | `git diff --check` | PASS |
| v2.0.2 榛勯噾鍩虹嚎 | `internal/compiler` legacy golden tests | PASS锛堥€愬瓧鑺備竴鑷达級 |

涓撻」娴嬭瘯锛坄internal/compiler/development_requirements_test.go` 绛夛級锛?
| # | 楠屾敹椤?| 娴嬭瘯 |
|---|---|---|
| 1 | `COMMIT_PROOF` / `RECONCILE` 涓嶅嚭鐜板湪 Host Admission | `TestDevelopmentRequirementClassSeparation`銆乣TestDevelopmentMutationAndReconcileRequirements` |
| 2 | Runtime Requirement 绮剧‘瑕嗙洊闈欐€?Mutation/Reconcile 濂戠害 | `TestDevelopmentRequirementClassSeparation`锛堟伆濂?1 鏉?COMMIT_PROOF + 1 鏉?RECONCILE锛屼綅缃畬鏁达級 |
| 3 | 鍚屼竴 Descriptor 涓ょ被淇′换瑕佹眰鍚屾椂瀛樺湪 | `TestDevelopmentEvidenceTrustRequirementsAreIndependent`锛坰ource-only / fact-only / both锛?|
| 4 | 澶?Independence Group銆佸 Root銆佸 Domain Pair | `TestDevelopmentIndependenceGroupsAreExact`锛? 缁?/ 2 鏍?/ 4 鍩?/ 7 瀵圭粍鍚堬紝閲嶆帓纭畾鎬э級+ `TestDevelopmentIndependenceSameDomainFails` |
| 5 | IngressCatalog Export 绮剧‘韬唤 | `TestDevelopmentIngressExportIdentity`锛堥敊璇?id銆侀敊璇矾寰勩€佺己澶?operation銆侀噸澶?operation銆侀敊璇?Owner銆佹湭瀵煎嚭 operation锛?|
| 6 | 鍒犻櫎浠绘剰 Requirement 鍚庨噸灏佸灞傛憳瑕佷粛 Verify FAIL | `TestDevelopmentVerifyRejectsRequirementClassMigration` |
| 7 | 淇敼 Procedure / Lock / Descriptor / Requirement 鍚?Verify FAIL | `TestDevelopmentVerifyRejectsResealedTampering`銆乣TestDevelopmentVerifyRejectsSourceChanges`銆乣TestDevelopmentVerifyRejectsRequirementClassMigration` |
| 8 | Native / WASM / Chromium 缁撴灉涓€鑷?| Chromium parity 闂ㄧ锛坆uild/status/diagnostics/artifactDigest/outputFiles 鍏ㄩ儴涓€鑷达級 |

CI锛歅R #51 鐨?checks 椤甸潰 <https://github.com/odango-chan/ca-skill-compiler/pull/51/checks>銆侾R #48 鐨勬枃妗ｄ笌鍥炬棤鍏?CI 瑙佸叾 checks 椤甸潰銆?
## 4. 宸茬煡闄愬埗

- MAS **G0 = NOT_RUN**锛堟病鏈夐€氳繃 G0 鐨勭湡瀹炵簿纭帴鍙ｅ揩鐓э級锛?*G1 = NOT_RUN**銆傛湰鎶ュ憡涓嶅绉?MAS 閫氳繃銆?- 鎵€鏈夋渚嬩粛涓?Compiler 鑷湁鐨勫悎鎴?Interface Package 蹇収锛屼笉浠ｈ〃鐪熷疄 Owner 鍙戝竷鎴栦骇鍝佽祫鏍笺€?- Development Artifact 鎭掍负 `NOT_ADMITTED`锛涚敓浜?Host 蹇呴』鎷掓敹 `DEVELOPMENT` profile銆?- 鐪熷疄 Commit Proof銆佸疄闄?Reconcile銆佽繍琛屾椂 Evidence銆丏eployment Grant銆丳rovider Binding 涓?Invocation 鎺堟潈浠嶇敱 Host/Runtime 璐熻矗锛屾湰瀹炵幇鍙褰曞叾闈欐€佷綅缃笌濂戠害銆?- v2.0.2 鍏ュ彛淇濇寔涓嶅彉锛涙湰鎶ュ憡涓嶆瀯鎴愬彂甯冩垨鍚堝苟鍐冲畾銆?
## 5. 鏈€缁堣亴璐ｇ煩闃碉紙鎽樿锛?
| 璐ｄ换 | Compiler | Interface Package | Host Admission | Runtime / Provider | Owner / Assurance Authority |
|---|---|---|---|---|---|
| Evidence 澹版槑/寮曠敤銆佸弬鏁扮粦瀹氥€丼ourcePath銆丼chema 绫诲瀷 | 鎵ц | 鎻愪緵蹇収 | 澶嶆牳 | 鈥?| 鈥?|
| EvidenceDescriptor 缁撴瀯涓?Producer/璺緞 | 鎵ц | 浣滀负姝ｅ紡瀵煎嚭鎵胯浇 | 澶嶆牳 | 鈥?| 鍙戝竷澹版槑 |
| Owner 鐪熷疄韬唤銆佺鍙戞潈闄?| 涓嶅垽瀹?| 澹版槑 `publisher` | 楠岃瘉 `OWNER_TRUST` | 鈥?| 鎵挎媴 |
| IndependentDomains / CommonFact 鎺堟潈 | 鍙鏌ュ０鏄庡舰鐘?| 鈥?| 楠岃瘉涓ょ被 TRUST | 鈥?| 鎺堟潈 |
| Independence 缁勯潤鎬佺粨鏋勪笌鍩熷鏋氫妇 | 鎵ц | 鎻愪緵 Descriptor | 楠岃瘉 `EVIDENCE_INDEPENDENCE_TRUST` | 鈥?| 鎺堟潈鍩熷 |
| Mutation / Commit Proof / Reconcile 闈欐€佸绾?| 鎵ц | 鎻愪緵 Callable/Schema | 澶嶆牳 | 瀹為檯鎵ц | 鈥?|
| 瀹為檯鎻愪氦銆佸疄闄呮仮澶嶇粨璁?| 涓嶄骇鐢熴€佷笉浼€?| 鈥?| 涓嶈冻浠ユ弧瓒?| 浜х敓骞堕獙璇?`COMMIT_PROOF` / `RECONCILE` | 鈥?|
| Deployment Grant / Provider Binding / Invocation 鎺堟潈 | 涓嶈鍙?| 鈥?| 楠岃瘉 | 鎵ц鏈熷鏍?| 閮ㄧ讲鏂圭鍙?|
| 浜х墿鎵ц鐘舵€?| 鍥哄畾 `NOT_ADMITTED` | 鈥?| 鎷掔粷鏈噯鍏ヤ骇鐗?| 浠呮墽琛屽凡鍑嗗叆 Invocation | 鈥?|

瀹屾暣鐭╅樀瑙?PR #48 鏋舵瀯鏂囨。 搂6銆?