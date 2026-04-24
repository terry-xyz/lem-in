package main

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"lem-in/internal/format"
)

//go:embed base.glb
var colonyModel []byte

type jsonRoom struct {
	Name    string  `json:"name"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Z       float64 `json:"z"`
	IsStart bool    `json:"isStart"`
	IsEnd   bool    `json:"isEnd"`
}

type jsonLink struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type jsonMovement struct {
	AntID    int    `json:"antId"`
	RoomName string `json:"room"`
}

type jsonData struct {
	AntCount  int              `json:"antCount"`
	Rooms     []jsonRoom       `json:"rooms"`
	Links     []jsonLink       `json:"links"`
	StartName string           `json:"startName"`
	EndName   string           `json:"endName"`
	Turns     [][]jsonMovement `json:"turns"`
	Error     string           `json:"error,omitempty"`
}

// main converts solver output from stdin into a self-contained HTML visualizer document.
func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: reading stdin: %v\n", err)
		os.Exit(1)
	}

	parsed, err := format.ParseOutput(string(input))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: parsing output: %v\n", err)
		os.Exit(1)
	}

	data := buildJSONData(parsed)

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: marshalling JSON: %v\n", err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: creating gzip writer: %v\n", err)
		os.Exit(1)
	}
	if _, err := gz.Write(colonyModel); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: compressing model: %v\n", err)
		os.Exit(1)
	}
	gz.Close()
	modelB64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	html := buildHTML(string(jsonBytes), modelB64)
	fmt.Print(html)
}

// bfsDepth assigns each room its shortest tunnel distance from the start so the 3D layout can stack layers by reachability.
func bfsDepth(rooms []format.ParsedRoom, links [][2]string, startName string) map[string]int {
	adj := make(map[string][]string)
	for _, link := range links {
		adj[link[0]] = append(adj[link[0]], link[1])
		adj[link[1]] = append(adj[link[1]], link[0])
	}

	depth := make(map[string]int)
	depth[startName] = 0
	queue := []string{startName}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, nb := range adj[curr] {
			if _, visited := depth[nb]; !visited {
				depth[nb] = depth[curr] + 1
				queue = append(queue, nb)
			}
		}
	}

	for _, r := range rooms {
		if _, ok := depth[r.Name]; !ok {
			depth[r.Name] = 0
		}
	}

	return depth
}

// colonyLayoutRadius picks a ring radius that fits the current depth layer inside the shell without collapsing sparse rooms into the center.
func colonyLayoutRadius(maxDepth, roomCount int, normalizedDepth float64) float64 {
	baseRadius := 2.8 * (0.5 + 0.5*math.Sin(normalizedDepth*math.Pi))
	if baseRadius < 0.8 {
		baseRadius = 0.8
	}

	targetHeight := math.Max(float64(maxDepth)*3.8, 6.0)
	colonyRadius := math.Max(1.25, targetHeight*0.12)
	if colonyRadius > 1.65 {
		colonyRadius = 1.65
	}

	desiredRadius := math.Max(baseRadius, float64(roomCount)*0.6)
	return math.Min(desiredRadius, colonyRadius)
}

// buildJSONData converts parsed solver output into the browser-friendly schema used by the embedded Three.js app.
func buildJSONData(parsed *format.ParsedOutput) jsonData {
	data := jsonData{
		AntCount:  parsed.AntCount,
		StartName: parsed.StartName,
		EndName:   parsed.EndName,
		Error:     parsed.Error,
	}

	if parsed.Error != "" {
		return data
	}

	depths := bfsDepth(parsed.Rooms, parsed.Links, parsed.StartName)

	maxDepth := 0
	for _, d := range depths {
		if d > maxDepth {
			maxDepth = d
		}
	}

	groups := make(map[int][]format.ParsedRoom)
	for _, r := range parsed.Rooms {
		d := depths[r.Name]
		groups[d] = append(groups[d], r)
	}

	ySpacing := 3.8

	for d := 0; d <= maxDepth; d++ {
		roomsAtLevel := groups[d]
		n := len(roomsAtLevel)
		if n == 0 {
			continue
		}

		// Depth controls both vertical height and shell radius so each BFS layer reads as a distinct ring.
		nd := float64(d) / math.Max(float64(maxDepth), 1.0)
		radius := colonyLayoutRadius(maxDepth, n, nd)

		for i, r := range roomsAtLevel {
			angle := 2.0 * math.Pi * float64(i) / float64(n)
			var nameHash float64
			for _, ch := range r.Name {
				nameHash += float64(ch)
			}
			// Hash-based jitter keeps rooms from lining up too perfectly while staying deterministic across renders.
			offset := math.Mod(nameHash*0.137, 0.3) - 0.15

			data.Rooms = append(data.Rooms, jsonRoom{
				Name:    r.Name,
				X:       radius * math.Cos(angle+offset),
				Y:       float64(d) * ySpacing,
				Z:       radius * math.Sin(angle+offset),
				IsStart: r.IsStart,
				IsEnd:   r.IsEnd,
			})
		}
	}

	for _, l := range parsed.Links {
		data.Links = append(data.Links, jsonLink{From: l[0], To: l[1]})
	}

	for _, turn := range parsed.Turns {
		var jTurn []jsonMovement
		for _, m := range turn {
			jTurn = append(jTurn, jsonMovement{AntID: m.AntID, RoomName: m.RoomName})
		}
		data.Turns = append(data.Turns, jTurn)
	}

	return data
}

// buildHTML returns the standalone HTML document with embedded simulation data, model bytes, styles, and viewer script.
func buildHTML(jsonStr, modelB64 string) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>lem-in Colony Cast</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
html,body{width:100%;height:100%;overflow:hidden;background:#f0ece6;
  font-family:Georgia,'Times New Roman',serif;color:#444}
canvas{display:block}
#loading{position:fixed;inset:0;background:#f0ece6;display:flex;
  align-items:center;justify-content:center;z-index:9999;
  transition:opacity 0.8s ease}
#loading span{font-size:0.85rem;color:#999;letter-spacing:0.15em;
  text-transform:uppercase}
#vignette{position:fixed;inset:0;pointer-events:none;z-index:98;
  background:radial-gradient(ellipse at center,transparent 60%,rgba(0,0,0,0.12) 100%)}
#info{position:fixed;top:24px;left:28px;z-index:100}
#info .title{font-size:1.05rem;color:#555;font-weight:600;
  letter-spacing:0.08em;text-transform:uppercase;margin-bottom:2px}
#info .detail{font-size:0.75rem;color:#999;line-height:1.5}
#turn{position:fixed;top:24px;right:28px;z-index:100;text-align:right}
#turn .num{font-size:1.2rem;color:#666;font-weight:600}
#turn .lbl{font-size:0.65rem;color:#aaa;text-transform:uppercase;
  letter-spacing:0.12em}
#controls{position:fixed;bottom:20px;left:50%;transform:translateX(-50%);
  background:rgba(35,32,28,0.82);backdrop-filter:blur(16px);
  -webkit-backdrop-filter:blur(16px);border-radius:14px;padding:12px 22px;
  display:flex;align-items:center;gap:14px;z-index:100;
  box-shadow:0 6px 30px rgba(0,0,0,0.2)}
#controls button{background:none;border:1px solid rgba(255,255,255,0.1);
  color:#aaa;padding:6px 14px;border-radius:7px;cursor:pointer;
  font-size:0.85rem;transition:all 0.2s}
#controls button:hover{border-color:rgba(255,255,255,0.3);color:#ddd}
#controls button.active{background:rgba(255,255,255,0.08);
  border-color:rgba(212,165,116,0.4);color:#d4a574}
.tl{display:flex;align-items:center;gap:8px;flex:1;min-width:120px}
.tl input{width:100%;-webkit-appearance:none;height:2px;
  background:rgba(255,255,255,0.12);border-radius:1px;outline:none}
.tl input::-webkit-slider-thumb{-webkit-appearance:none;width:12px;
  height:12px;background:#d4a574;border-radius:50%;cursor:pointer}
.sp{display:flex;align-items:center;gap:6px;font-size:0.7rem;color:#777}
.sp input{width:65px;-webkit-appearance:none;height:2px;
  background:rgba(255,255,255,0.12);border-radius:1px;outline:none}
.sp input::-webkit-slider-thumb{-webkit-appearance:none;width:10px;
  height:10px;background:#d4a574;border-radius:50%;cursor:pointer}
#colony-toggle-wrap{position:fixed;right:28px;bottom:20px;z-index:101;
  animation:fadeIn 1s ease-out 0.3s both}
#auto-rotate-toggle-wrap{position:fixed;left:28px;bottom:20px;z-index:101;
  animation:fadeIn 1s ease-out 0.3s both}
@media (max-width: 820px) {
  #auto-rotate-toggle-wrap{left:16px;bottom:82px}
}
.button-wrap{transition:all 400ms cubic-bezier(0.25,1,0.5,1)}
.glass-button{position:relative;border:none;border-radius:999px;cursor:pointer;
  outline:none;background:linear-gradient(-75deg,rgba(255,255,255,0.05),
  rgba(255,255,255,0.2),rgba(255,255,255,0.05));
  box-shadow:inset 0 2px 2px rgba(0,0,0,0.05),
  inset 0 -2px 2px rgba(255,255,255,0.5),0 4px 2px -2px rgba(0,0,0,0.2),
  0 0 2px 4px rgba(255,255,255,0.2) inset;
  backdrop-filter:blur(4px);-webkit-backdrop-filter:blur(4px);
  transition:all 400ms cubic-bezier(0.25,1,0.5,1)}
.glass-button:hover{transform:scale(0.975);backdrop-filter:blur(1px);
  -webkit-backdrop-filter:blur(1px);box-shadow:inset 0 2px 2px rgba(0,0,0,0.05),
  inset 0 -2px 2px rgba(255,255,255,0.5),0 3px 1px -2px rgba(0,0,0,0.25),
  0 0 1px 2px rgba(255,255,255,0.5) inset}
.glass-button:active{transform:scale(0.95) rotate3d(1,0,0,25deg);
  box-shadow:inset 0 2px 2px rgba(0,0,0,0.05),
  inset 0 -2px 2px rgba(255,255,255,0.5),0 2px 2px -2px rgba(0,0,0,0.2),
  0 0 2px 4px rgba(255,255,255,0.2) inset,0 4px 1px 0 rgba(0,0,0,0.05),
  0 4px 0 0 rgba(255,255,255,0.75),inset 0 4px 1px 0 rgba(0,0,0,0.15)}
.glass-button::after{content:'';position:absolute;inset:-1px;border-radius:999px;
  padding:1px;box-sizing:border-box;background:
  conic-gradient(from -75deg at 50% 50%,rgba(0,0,0,0.5),rgba(0,0,0,0) 5% 40%,
  rgba(0,0,0,0.5) 50%,rgba(0,0,0,0) 60% 95%,rgba(0,0,0,0.5)),
  linear-gradient(180deg,rgba(255,255,255,0.5),rgba(255,255,255,0.5));
  -webkit-mask:linear-gradient(#000 0 0) content-box,linear-gradient(#000 0 0);
  -webkit-mask-composite:xor;mask:linear-gradient(#000 0 0) content-box,
  linear-gradient(#000 0 0);mask-composite:exclude;
  box-shadow:inset 0 0 0 0.5px rgba(255,255,255,0.5);
  transition:all 400ms cubic-bezier(0.25,1,0.5,1)}
.button-text{position:relative;display:block;user-select:none;font-weight:600;
  font-size:0.72rem;color:#3f3a33;letter-spacing:0.12em;padding:13px 22px;
  text-shadow:0 4px 1px rgba(0,0,0,0.1);
  transition:all 400ms cubic-bezier(0.25,1,0.5,1)}
.glass-button:hover .button-text{text-shadow:1px 1px 1px rgba(0,0,0,0.12)}
.glass-button:active .button-text{text-shadow:1px 4px 1px rgba(0,0,0,0.12)}
.button-shine{position:absolute;inset:0.5px;border-radius:999px;
  background:linear-gradient(-45deg,rgba(255,255,255,0) 0%,
  rgba(255,255,255,0.5) 40% 50%,rgba(255,255,255,0) 55%);
  mix-blend-mode:screen;pointer-events:none;background-size:200% 200%;
  background-position:0% 50%;background-repeat:no-repeat;
  transition:background-position 500ms cubic-bezier(0.25,1,0.5,1)}
.glass-button:hover .button-shine{background-position:25% 50%}
.glass-button:active .button-shine{background-position:50% 15%}
#hover{position:fixed;padding:8px 14px;background:rgba(35,32,28,0.9);
  backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);
  border-radius:8px;font-size:0.72rem;color:#ccc;pointer-events:none;
  display:none;white-space:pre-line;line-height:1.5;z-index:200}
@keyframes fadeIn{from{opacity:0}to{opacity:1}}
</style>
</head>
<body>
<div id="loading"><span>Loading colony...</span></div>
<div id="vignette"></div>
<div id="info">
  <div class="title">Colony</div>
  <div class="detail" id="info-detail">Loading...</div>
</div>
<div id="turn">
  <div class="num" id="turn-value">0 / 0</div>
  <div class="lbl">Turn</div>
</div>
<div id="controls">
  <button id="btn-restart" title="Restart">&#8634;</button>
  <button id="btn-prev" title="Previous">&#9664;</button>
  <button id="btn-play" class="active" title="Play/Pause">&#10074;&#10074;</button>
  <button id="btn-next" title="Next">&#9654;</button>
  <div class="tl">
    <input type="range" id="tl-slider" min="0" max="0" step="1" value="0">
  </div>
  <div class="sp">
    <span>Speed</span>
    <input type="range" id="sp-slider" min="0.2" max="4" step="0.1" value="1">
    <span id="sp-val">1.0x</span>
  </div>
</div>
<div id="colony-toggle-wrap" class="button-wrap">
  <button id="btn-show-colony" class="glass-button" type="button" title="Toggle colony shell visibility">
    <span class="button-text" id="btn-show-colony-text">SHOW COLONY: YES</span>
    <div class="button-shine"></div>
  </button>
</div>
<div id="auto-rotate-toggle-wrap" class="button-wrap">
  <button id="btn-auto-rotate" class="glass-button" type="button" title="Toggle auto-rotation" aria-pressed="true">
    <span class="button-text" id="btn-auto-rotate-text">AUTO-ROTATE: YES</span>
    <div class="button-shine"></div>
  </button>
</div>
<div id="hover"></div>

<script>
var SIM_DATA = `)
	sb.WriteString(jsonStr)
	sb.WriteString(`;
var COLONY_MODEL_GZ_B64 = "`)
	sb.WriteString(modelB64)
	sb.WriteString(`";
</script>

<script type="importmap">
{
  "imports": {
    "three": "https://cdn.jsdelivr.net/npm/three@0.166.0/build/three.module.js",
    "three/addons/": "https://cdn.jsdelivr.net/npm/three@0.166.0/examples/jsm/"
  }
}
</script>

<script type="module">
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js';

// ---------- ERROR HANDLING ----------
if (SIM_DATA.error) {
  document.getElementById('loading').style.display = 'none';
  var ov = document.createElement('div');
  ov.style.cssText = 'position:fixed;inset:0;background:#f0ece6;display:flex;align-items:center;justify-content:center;flex-direction:column;z-index:1000';
  ov.innerHTML = '<h1 style="color:#c44;font-size:1.5rem;margin-bottom:1rem">Error</h1><p style="color:#888;max-width:500px;text-align:center">' + SIM_DATA.error.replace(/</g,'&lt;') + '</p>';
  document.body.appendChild(ov);
  throw new Error('Input error');
}

// ---------- NOISE ----------
function hash3(x, y, z) {
  var s = Math.sin(x * 127.1 + y * 311.7 + z * 74.7) * 43758.5453;
  return s - Math.floor(s);
}

function noise3d(x, y, z) {
  var ix = Math.floor(x), iy = Math.floor(y), iz = Math.floor(z);
  var fx = x - ix, fy = y - iy, fz = z - iz;
  var ux = fx*fx*(3-2*fx), uy = fy*fy*(3-2*fy), uz = fz*fz*(3-2*fz);
  var a = hash3(ix,iy,iz), b = hash3(ix+1,iy,iz);
  var c = hash3(ix,iy+1,iz), d = hash3(ix+1,iy+1,iz);
  var e = hash3(ix,iy,iz+1), f = hash3(ix+1,iy,iz+1);
  var g = hash3(ix,iy+1,iz+1), h = hash3(ix+1,iy+1,iz+1);
  var x0 = a+(b-a)*ux, x1 = c+(d-c)*ux, x2 = e+(f-e)*ux, x3 = g+(h-g)*ux;
  var y0 = x0+(x1-x0)*uy, y1 = x2+(x3-x2)*uy;
  return y0+(y1-y0)*uz;
}

function fbm(x, y, z) {
  var v=0, a=1, fr=1, m=0;
  for (var i=0; i<4; i++) { v+=noise3d(x*fr,y*fr,z*fr)*a; m+=a; a*=0.5; fr*=2; }
  return v/m;
}

// ---------- SCENE ----------
var scene = new THREE.Scene();
scene.background = new THREE.Color(0xf5f2ec);
var camera = new THREE.PerspectiveCamera(45, window.innerWidth/window.innerHeight, 0.1, 500);
var renderer = new THREE.WebGLRenderer({ antialias: true });
renderer.setSize(window.innerWidth, window.innerHeight);
renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
renderer.toneMapping = THREE.ACESFilmicToneMapping;
renderer.toneMappingExposure = 1.1;
renderer.shadowMap.enabled = true;
renderer.shadowMap.type = THREE.PCFSoftShadowMap;
document.body.prepend(renderer.domElement);

var controls = new OrbitControls(camera, renderer.domElement);
controls.enableDamping = true;
controls.dampingFactor = 0.08;
controls.rotateSpeed = 0.5;
controls.zoomSpeed = 0.8;
controls.minDistance = 5;
controls.maxDistance = 200;
controls.autoRotate = true;
controls.autoRotateSpeed = 0.4;

// ---------- DATA ----------
var rooms = SIM_DATA.rooms || [];
var links = SIM_DATA.links || [];
var turns = SIM_DATA.turns || [];
var roomMap = {};
var cx=0, cy=0, cz=0, minY=Infinity, maxY=-Infinity;
rooms.forEach(function(r) { cx+=r.x; cy+=r.y; cz+=r.z; if(r.y<minY)minY=r.y; if(r.y>maxY)maxY=r.y; });
if (rooms.length > 0) { cx/=rooms.length; cy/=rooms.length; cz/=rooms.length; }
var totalHeight = maxY - minY;
var maxR = 0;
rooms.forEach(function(r) {
  var dx = r.x-cx, dz = r.z-cz;
  var dd = Math.sqrt(dx*dx+dz*dz);
  if (dd > maxR) maxR = dd;
});
rooms.forEach(function(r) {
  roomMap[r.name] = { pos: new THREE.Vector3(r.x-cx, r.y-cy, r.z-cz), data: r };
});

// ---------- ENVIRONMENT MAP ----------
var pmrem = new THREE.PMREMGenerator(renderer);
var envSc = new THREE.Scene();
var envGeo = new THREE.SphereGeometry(20, 32, 16);
var envMat = new THREE.ShaderMaterial({
  side: THREE.BackSide,
  uniforms: {
    topC: { value: new THREE.Color(0xf5f0e8) },
    botC: { value: new THREE.Color(0xc8c0b5) }
  },
  vertexShader: 'varying vec3 vP; void main(){ vP = position; gl_Position = projectionMatrix * modelViewMatrix * vec4(position,1.0); }',
  fragmentShader: 'uniform vec3 topC; uniform vec3 botC; varying vec3 vP; void main(){ float h = normalize(vP).y * 0.5 + 0.5; gl_FragColor = vec4(mix(botC, topC, h), 1.0); }'
});
envSc.add(new THREE.Mesh(envGeo, envMat));
var spotGeo = new THREE.SphereGeometry(2, 16, 8);
var spotMat = new THREE.MeshBasicMaterial({ color: 0xffffff });
var spotMesh = new THREE.Mesh(spotGeo, spotMat);
spotMesh.position.set(8, 12, 6);
envSc.add(spotMesh);
var envMap = pmrem.fromScene(envSc, 0, 0.1, 100).texture;
scene.environment = envMap;
pmrem.dispose();

// ---------- LIGHTING ----------
var keyLight = new THREE.SpotLight(0xfff8ee, 6, 200, Math.PI/5, 0.3);
keyLight.position.set(10, totalHeight + 8, 10);
keyLight.castShadow = true;
keyLight.shadow.mapSize.set(2048, 2048);
keyLight.shadow.bias = -0.001;
keyLight.target.position.set(0, 0, 0);
scene.add(keyLight);
scene.add(keyLight.target);

var fillLight = new THREE.DirectionalLight(0xd0dce8, 0.8);
fillLight.position.set(-10, 6, -4);
scene.add(fillLight);

var rimLight = new THREE.DirectionalLight(0xffffff, 0.35);
rimLight.position.set(0, 4, -12);
scene.add(rimLight);

var hemiLight = new THREE.HemisphereLight(0xf0ece6, 0x0a0805, 0.4);
scene.add(hemiLight);

// ---------- LOAD COLONY MODEL ----------
async function loadColonyModel() {
  var binaryStr = atob(COLONY_MODEL_GZ_B64);
  var len = binaryStr.length;
  var bytes = new Uint8Array(len);
  for (var i = 0; i < len; i++) bytes[i] = binaryStr.charCodeAt(i);

  var ds = new DecompressionStream('gzip');
  var blob = new Blob([bytes]);
  var decompressed = await new Response(blob.stream().pipeThrough(ds)).arrayBuffer();

  return new Promise(function(resolve, reject) {
    new GLTFLoader().parse(decompressed, '', resolve, reject);
  });
}

var gltf = await loadColonyModel();
var colony = gltf.scene;
var showColony = true;
var autoRotateEnabled = true;
var colonyShellMaterials = [];
var colonyVisibleOpacity = 1.0;
var colonyHiddenOpacity = 0.0;
var groundVisibleOpacity = 0.15;
var groundHiddenOpacity = 0.0;

// Apply cast-metal material
var colonyMat = new THREE.MeshPhysicalMaterial({
  color: 0xccc8c0,
  metalness: 0.95,
  roughness: 0.14,
  clearcoat: 0.35,
  clearcoatRoughness: 0.12,
  envMapIntensity: 2.2,
  sheen: 0.15,
  sheenColor: new THREE.Color(0xffffff)
});
var baseMat = new THREE.MeshPhysicalMaterial({
  color: 0x111111,
  metalness: 0.05,
  roughness: 0.9,
  envMapIntensity: 0.3
});

// Scale and position colony
var modelBox = new THREE.Box3().setFromObject(colony);
var modelCenter = modelBox.getCenter(new THREE.Vector3());
var modelSize = modelBox.getSize(new THREE.Vector3());
var targetH = Math.max(totalHeight, 6);
var sc = targetH / modelSize.y * 1.15;
var roomFloorY = minY - cy;
colony.scale.setScalar(sc);
colony.position.set(-modelCenter.x * sc, -modelCenter.y * sc, -modelCenter.z * sc);
scene.add(colony);
colony.updateMatrixWorld(true);

// Recompute bounds after scaling and align the colony floor to the room floor
var scaledBox = new THREE.Box3().setFromObject(colony);
var scaledSize = scaledBox.getSize(new THREE.Vector3());
var scaledMinY = scaledBox.min.y;
var colonyLift = roomFloorY - scaledMinY;
colony.position.y += colonyLift;
colony.updateMatrixWorld(true);

var alignedBox = new THREE.Box3().setFromObject(colony);
var alignedSize = alignedBox.getSize(new THREE.Vector3());
var groundY = alignedBox.min.y;
var baseThresholdWorld = groundY + alignedSize.y * 0.05;

// Pre-compute room world positions for shader
var roomPosArray = [];
rooms.forEach(function(r) { roomPosArray.push(roomMap[r.name].pos.clone()); });
var maxRooms = Math.max(roomPosArray.length, 1);

// Apply materials with per-vertex cave darkening
var caveR = 3.0;
colony.traverse(function(child) {
  if (child.isMesh) {
    if (child.geometry && child.geometry.attributes.position) {
      var pos = child.geometry.attributes.position;
      var norm = child.geometry.attributes.normal;
      var colors = new Float32Array(pos.count * 3);
      var colonyColor = new THREE.Color(0xccc8c0);
      var blackColor = new THREE.Color(0x111111);
      var v = new THREE.Vector3();
      var n = new THREE.Vector3();
      var normalMatrix = new THREE.Matrix3().getNormalMatrix(child.matrixWorld);

      for (var vi = 0; vi < pos.count; vi++) {
        v.set(pos.getX(vi), pos.getY(vi), pos.getZ(vi));
        v.applyMatrix4(child.matrixWorld);
        var c = v.y <= baseThresholdWorld ? blackColor : colonyColor;

        if (v.y > baseThresholdWorld && norm) {
          n.set(norm.getX(vi), norm.getY(vi), norm.getZ(vi));
          n.applyMatrix3(normalMatrix).normalize();

          var maxDark = 0;
          for (var ri = 0; ri < roomPosArray.length; ri++) {
            var rp = roomPosArray[ri];
            var dx = v.x - rp.x, dy = v.y - rp.y, dz = v.z - rp.z;
            if (Math.abs(dy) > 2.0) continue;
            var dist = Math.sqrt(dx*dx + dy*dy + dz*dz);
            if (dist > caveR || dist < 0.01) continue;

            // Direction from vertex to room (captures ceilings/floors too)
            var toRX = -dx/dist, toRY = -dy/dist, toRZ = -dz/dist;
            var facing = n.x*toRX + n.y*toRY + n.z*toRZ;
            if (facing < 0.2) continue;

            var facingMask = Math.min((facing - 0.2) / 0.4, 1.0);
            var distInf = 1.0 - dist / caveR;

            // Depth: vertices closer to room center are deeper in cave
            var depthFactor = distInf * distInf; // quadratic — shallow=faint, deep=strong
            var dark = facingMask * depthFactor;
            if (dark > maxDark) maxDark = dark;
          }

          var f = 1.0 - maxDark;
          colors[vi*3] = c.r * f;
          colors[vi*3+1] = c.g * f;
          colors[vi*3+2] = c.b * f;
        } else {
          colors[vi*3] = c.r;
          colors[vi*3+1] = c.g;
          colors[vi*3+2] = c.b;
        }
      }
      child.geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));
    }

    var mat = new THREE.MeshPhysicalMaterial({
      vertexColors: true,
      metalness: 0.95,
      roughness: 0.14,
      clearcoat: 0.35,
      clearcoatRoughness: 0.12,
      envMapIntensity: 2.2,
      sheen: 0.15,
      sheenColor: new THREE.Color(0xffffff),
      transparent: true,
      opacity: colonyVisibleOpacity
    });

    // Aggressively dim ALL light in dark cave areas (overcomes clearcoat/env washout)
    mat.onBeforeCompile = function(shader) {
      shader.fragmentShader = shader.fragmentShader.replace(
        '#include <dithering_fragment>',
        'float vBright = (vColor.r + vColor.g + vColor.b) / 3.0;\n' +
        'float isBase = step(vBright, 0.08);\n' +
        'float darkAmt = smoothstep(0.72, 0.20, vBright);\n' +
        'float dimFactor = mix(1.0, 0.04, darkAmt * darkAmt);\n' +
        'gl_FragColor.rgb *= mix(dimFactor, 1.0, isBase);\n' +
        '#include <dithering_fragment>'
      );
    };

    child.material = mat;
    colonyShellMaterials.push(mat);
    child.castShadow = true;
    child.receiveShadow = true;
  }
});

// ---------- TUNNELS ----------
links.forEach(function(l) {
  var from = roomMap[l.from], to = roomMap[l.to];
  if (!from || !to) return;
  // Organic midpoint displacement for natural-looking tunnels
  var mid = new THREE.Vector3().lerpVectors(from.pos, to.pos, 0.5);
  var dist = from.pos.distanceTo(to.pos);
  mid.x += (fbm(mid.x, mid.y, mid.z) - 0.5) * dist * 0.15;
  mid.z += (fbm(mid.x + 50, mid.y, mid.z) - 0.5) * dist * 0.15;
  var curve = new THREE.CatmullRomCurve3([from.pos.clone(), mid, to.pos.clone()]);
  var g = new THREE.TubeGeometry(curve, 12, 0.14, 8, false);
  var m = new THREE.MeshStandardMaterial({
    color: 0x4a3d30, metalness: 0.1, roughness: 0.85,
    transparent: true, opacity: 0.6,
    emissive: 0x2a1a0f, emissiveIntensity: 0.08
  });
  var mesh = new THREE.Mesh(g, m);
  scene.add(mesh);
});

// ---------- GROUND SHADOW ----------
var groundGeo = new THREE.PlaneGeometry(120, 120);
var groundMat = new THREE.ShadowMaterial({ opacity: 0.15 });
var groundMesh = new THREE.Mesh(groundGeo, groundMat);
groundMesh.rotation.x = -Math.PI / 2;
groundMesh.position.y = groundY - 0.02;
groundMesh.receiveShadow = true;
scene.add(groundMesh);

function applyColonyVisibility() {
  var shellOpacity = showColony ? colonyVisibleOpacity : colonyHiddenOpacity;
  var shadowOpacity = showColony ? groundVisibleOpacity : groundHiddenOpacity;

  colony.visible = showColony;
  groundMesh.visible = showColony;

  colonyShellMaterials.forEach(function(mat) {
    mat.opacity = shellOpacity;
    mat.transparent = true;
    mat.needsUpdate = true;
  });

  groundMat.opacity = shadowOpacity;
  groundMat.transparent = true;
  groundMat.needsUpdate = true;
  btnShowColonyText.textContent = showColony ? 'SHOW COLONY: YES' : 'SHOW COLONY: OFF';
}

function applyAutoRotateState() {
  controls.autoRotate = autoRotateEnabled;
  btnAutoRotate.setAttribute('aria-pressed', autoRotateEnabled ? 'true' : 'false');
  btnAutoRotateText.textContent = autoRotateEnabled ? 'AUTO-ROTATE: YES' : 'AUTO-ROTATE: OFF';
}

// ---------- CAMERA ----------
var camDist = Math.max(totalHeight * 2.0, alignedSize.length() * 1.5, 22);
var cameraFloorOffset = 0.18;
camera.position.set(camDist * 0.55, totalHeight * 0.5, camDist * 0.55);
controls.target.set(0, 0, 0);
camera.lookAt(0, 0, 0);
var initialCameraPosition = camera.position.clone();
var initialControlsTarget = controls.target.clone();
var cameraResetActive = false;
var cameraResetElapsed = 0;
var cameraResetDuration = 0.6;
var cameraResetFromPosition = new THREE.Vector3();
var cameraResetFromTarget = new THREE.Vector3();

function startCameraResetToOpeningView() {
  cameraResetActive = true;
  cameraResetElapsed = 0;
  cameraResetFromPosition.copy(camera.position);
  cameraResetFromTarget.copy(controls.target);
}

// ---------- ANT MANAGEMENT ----------
var antMeshes = {};
var antCurrentRoom = {};

function getOrCreateAnt(antId) {
  if (antMeshes[antId]) return antMeshes[antId];
  // Dark ant body — capsule shape for elongated insect look
  var bodyCol = new THREE.Color(0x2a1a0a);
  var g = new THREE.CapsuleGeometry(0.07, 0.18, 4, 8);
  var m = new THREE.MeshStandardMaterial({
    color: bodyCol, metalness: 0.3, roughness: 0.7,
    emissive: bodyCol, emissiveIntensity: 0.05
  });
  var mesh = new THREE.Mesh(g, m);
  var occlusionOutline = new THREE.MeshBasicMaterial({
    color: 0xffd36a,
    transparent: true,
    opacity: 0.38,
    depthTest: true,
    depthWrite: false,
    depthFunc: THREE.GreaterDepth,
    side: THREE.BackSide
  });
  var outline = new THREE.Mesh(g.clone(), occlusionOutline);
  outline.scale.setScalar(1.9);
  outline.renderOrder = 20;
  outline.visible = true;
  mesh.add(outline);
  mesh.userData.outline = outline;
  mesh.visible = false;
  mesh.renderOrder = 10;
  scene.add(mesh);
  antMeshes[antId] = mesh;
  antCurrentRoom[antId] = SIM_DATA.startName;
  return mesh;
}

function showAnt(mesh) {
  mesh.visible = true;
  if (mesh.userData.outline) mesh.userData.outline.visible = true;
}

// ---------- TURN ANIMATIONS ----------
function buildTurnAnims() {
  var anims = [];
  var antPos = {};
  for (var t = 0; t < turns.length; t++) {
    var turnAnims = [];
    for (var j = 0; j < turns[t].length; j++) {
      var mv = turns[t][j];
      var fromName = antPos[mv.antId] || SIM_DATA.startName;
      var toName = mv.room;
      var fromR = roomMap[fromName];
      var toR = roomMap[toName];
      if (fromR && toR) {
        turnAnims.push({
          antId: mv.antId,
          fromPos: fromR.pos.clone(),
          toPos: toR.pos.clone(),
          fromName: fromName,
          toName: toName
        });
      }
      antPos[mv.antId] = toName;
    }
    anims.push(turnAnims);
  }
  return anims;
}

var turnAnimations = buildTurnAnims();

function getAnimPos(from, to, t) {
  var m1 = new THREE.Vector3().lerpVectors(from, to, 0.33);
  var m2 = new THREE.Vector3().lerpVectors(from, to, 0.67);
  var off = 0.45;
  m1.x += (fbm(m1.x, m1.y, m1.z) - 0.5) * off;
  m1.z += (fbm(m1.x+50, m1.y, m1.z) - 0.5) * off;
  m2.x += (fbm(m2.x+100, m2.y, m2.z) - 0.5) * off;
  m2.z += (fbm(m2.x, m2.y+100, m2.z) - 0.5) * off;
  var curve = new THREE.CatmullRomCurve3([from.clone(), m1, m2, to.clone()]);
  var et = t < 0.5 ? 2*t*t : 1 - Math.pow(-2*t+2, 2)/2;
  return curve.getPoint(et);
}

// ---------- ANIMATION STATE ----------
var currentTurn = 0;
var turnProgress = 0;
var isPlaying = true;
var speed = 1.0;
var animComplete = turns.length === 0;

function animateAnts(turnIdx, progress) {
  if (turnIdx >= turnAnimations.length) return;
  var anims = turnAnimations[turnIdx];
  var up = new THREE.Vector3(0, 1, 0);
  for (var i = 0; i < anims.length; i++) {
    var a = anims[i];
    var mesh = getOrCreateAnt(a.antId);
    showAnt(mesh);
    var pos = getAnimPos(a.fromPos, a.toPos, progress);
    mesh.position.copy(pos);
    // Orient capsule along movement direction
    var nextPos = getAnimPos(a.fromPos, a.toPos, Math.min(progress + 0.02, 1.0));
    var dir = new THREE.Vector3().subVectors(nextPos, pos).normalize();
    if (dir.lengthSq() > 0.0001) {
      mesh.quaternion.setFromUnitVectors(up, dir);
    }
  }
}

function snapAnts(turnIdx) {
  if (turnIdx >= turnAnimations.length) return;
  var anims = turnAnimations[turnIdx];
  for (var i = 0; i < anims.length; i++) {
    var a = anims[i];
    var mesh = getOrCreateAnt(a.antId);
    mesh.position.copy(a.toPos);
    showAnt(mesh);
    antCurrentRoom[a.antId] = a.toName;
  }
}

function hideAllAnts() {
  Object.keys(antMeshes).forEach(function(k) {
    antMeshes[k].visible = false;
    if (antMeshes[k].userData.outline) antMeshes[k].userData.outline.visible = false;
  });
}

function rebuildAntsToTurn(targetTurn) {
  hideAllAnts();
  var antPos = {};
  for (var t = 0; t < targetTurn; t++) {
    for (var j = 0; j < turns[t].length; j++) {
      antPos[turns[t][j].antId] = turns[t][j].room;
    }
  }
  Object.keys(antPos).forEach(function(idStr) {
    var id = parseInt(idStr);
    var roomName = antPos[id];
    var mesh = getOrCreateAnt(id);
    var room = roomMap[roomName];
    if (room && roomName !== SIM_DATA.endName) {
      mesh.position.copy(room.pos);
      showAnt(mesh);
    }
  });
}

function hideFinishedAnts() {
  var finalRoom = {};
  for (var t = 0; t < turns.length; t++) {
    for (var j = 0; j < turns[t].length; j++) {
      finalRoom[turns[t][j].antId] = turns[t][j].room;
    }
  }
  Object.keys(finalRoom).forEach(function(idStr) {
    if (finalRoom[idStr] === SIM_DATA.endName) {
      var mesh = antMeshes[parseInt(idStr)];
      if (mesh) mesh.visible = false;
    }
  });
}

// ---------- ANT PATHS (for hover) ----------
var antPaths = {};
(function() {
  for (var t = 0; t < turns.length; t++) {
    for (var j = 0; j < turns[t].length; j++) {
      var mv = turns[t][j];
      if (!antPaths[mv.antId]) antPaths[mv.antId] = [SIM_DATA.startName];
      antPaths[mv.antId].push(mv.room);
    }
  }
})();

// ---------- RAYCASTING ----------
var raycaster = new THREE.Raycaster();
var mouse = new THREE.Vector2();
var hoverLabel = document.getElementById('hover');
var roomSpheres = [];

rooms.forEach(function(r) {
  var p = roomMap[r.name].pos;
  var g = new THREE.SphereGeometry(0.8, 8, 6);
  var m = new THREE.MeshBasicMaterial({ visible: false });
  var mesh = new THREE.Mesh(g, m);
  mesh.position.copy(p);
  mesh.userData = { roomName: r.name, isStart: r.isStart, isEnd: r.isEnd };
  scene.add(mesh);
  roomSpheres.push(mesh);
});

renderer.domElement.addEventListener('mousemove', function(ev) {
  mouse.x = (ev.clientX / window.innerWidth) * 2 - 1;
  mouse.y = -(ev.clientY / window.innerHeight) * 2 + 1;
  raycaster.setFromCamera(mouse, camera);
  var hits = raycaster.intersectObjects(roomSpheres, false);
  if (hits.length > 0) {
    var hit = hits[0].object;
    var rName = hit.userData.roomName;
    hoverLabel.style.display = 'block';
    hoverLabel.style.left = (ev.clientX + 15) + 'px';
    hoverLabel.style.top = (ev.clientY - 10) + 'px';
    var nb = [];
    links.forEach(function(l) {
      if (l.from === rName) nb.push(l.to);
      else if (l.to === rName) nb.push(l.from);
    });
    var ants = [];
    Object.keys(antPaths).forEach(function(idStr) {
      if (antPaths[idStr].indexOf(rName) !== -1) ants.push(idStr);
    });
    var line1 = rName;
    if (hit.userData.isStart) line1 += '  [START]';
    if (hit.userData.isEnd) line1 += '  [END]';
    var line2 = '';
    if (hit.userData.isStart) line2 = 'All ' + SIM_DATA.antCount + ' ants depart';
    else if (hit.userData.isEnd) line2 = 'All ' + SIM_DATA.antCount + ' ants arrive';
    else if (ants.length > 0) {
      var ids = ants.length <= 6 ? ants.join(', ') : ants.slice(0,5).join(', ') + ' +' + (ants.length-5);
      line2 = 'Ants: ' + ids;
    } else line2 = 'No traffic';
    var nbStr = nb.length <= 4 ? nb.join(', ') : nb.slice(0,3).join(', ') + ' +' + (nb.length-3);
    hoverLabel.textContent = line1 + '\n' + line2 + '\nLinks: ' + nbStr;
  } else {
    hoverLabel.style.display = 'none';
  }
});

// ---------- UI CONTROLS ----------
var btnPlay = document.getElementById('btn-play');
var btnPrev = document.getElementById('btn-prev');
var btnNext = document.getElementById('btn-next');
var btnRestart = document.getElementById('btn-restart');
var btnShowColony = document.getElementById('btn-show-colony');
var btnShowColonyText = document.getElementById('btn-show-colony-text');
var btnAutoRotate = document.getElementById('btn-auto-rotate');
var btnAutoRotateText = document.getElementById('btn-auto-rotate-text');
var spSlider = document.getElementById('sp-slider');
var spVal = document.getElementById('sp-val');
var tlSlider = document.getElementById('tl-slider');
var turnValue = document.getElementById('turn-value');
var infoDetail = document.getElementById('info-detail');

function applyPlaybackState() {
  btnPlay.innerHTML = isPlaying ? '&#10074;&#10074;' : '&#9654;';
  btnPlay.classList.toggle('active', isPlaying);
}

infoDetail.textContent = SIM_DATA.antCount + ' ants  |  ' + rooms.length + ' rooms  |  ' + links.length + ' tunnels  |  ' + turns.length + ' turns';
tlSlider.max = turns.length;
applyPlaybackState();

btnPlay.addEventListener('click', function() {
  isPlaying = !isPlaying;
  applyPlaybackState();
});
btnRestart.addEventListener('click', function() {
  currentTurn = 0; turnProgress = 0;
  animComplete = turns.length === 0;
  hideAllAnts();
});
btnPrev.addEventListener('click', function() {
  if (currentTurn > 0) { currentTurn--; turnProgress = 0; animComplete = false; rebuildAntsToTurn(currentTurn); }
});
btnNext.addEventListener('click', function() {
  if (currentTurn < turns.length - 1) { currentTurn++; turnProgress = 0; animComplete = false; rebuildAntsToTurn(currentTurn); }
});
btnShowColony.addEventListener('click', function() {
  showColony = !showColony;
  applyColonyVisibility();
  if (!showColony) cameraResetActive = false;
  if (showColony && camera.position.y < groundY) {
    startCameraResetToOpeningView();
  }
});
btnAutoRotate.addEventListener('click', function() {
  autoRotateEnabled = !autoRotateEnabled;
  applyAutoRotateState();
});
spSlider.addEventListener('input', function() {
  speed = parseFloat(spSlider.value);
  spVal.textContent = speed.toFixed(1) + 'x';
});
tlSlider.addEventListener('input', function() {
  var target = parseInt(tlSlider.value);
  currentTurn = target; turnProgress = 0;
  if (target >= turns.length) { animComplete = true; rebuildAntsToTurn(turns.length); hideFinishedAnts(); }
  else { animComplete = false; rebuildAntsToTurn(target); }
});
applyColonyVisibility();
applyAutoRotateState();

// ---------- LOADING FADE ----------
var ld = document.getElementById('loading');
if (ld) { ld.style.opacity = '0'; setTimeout(function(){ ld.remove(); }, 800); }

// ---------- ANIMATION LOOP ----------
var clock = new THREE.Clock();

function animate() {
  requestAnimationFrame(animate);
  var dt = clock.getDelta();

  if (isPlaying && !animComplete && turns.length > 0 && currentTurn < turns.length) {
    turnProgress += dt * speed * 0.8;
    if (turnProgress >= 1.0) {
      snapAnts(currentTurn);
      currentTurn++;
      turnProgress = 0;
      if (currentTurn >= turns.length) { animComplete = true; hideFinishedAnts(); }
    } else {
      animateAnts(currentTurn, turnProgress);
    }
  }

  var displayTurn = animComplete ? turns.length : currentTurn + (turns.length > 0 ? 1 : 0);
  turnValue.textContent = displayTurn + ' / ' + turns.length;
  tlSlider.value = animComplete ? turns.length : currentTurn;

  controls.update();
  if (cameraResetActive) {
  cameraResetElapsed += dt;
  var resetT = Math.min(cameraResetElapsed / cameraResetDuration, 1.0);
  resetT = resetT < 0.5 ? 4 * resetT * resetT * resetT : 1 - Math.pow(-2 * resetT + 2, 3) / 2;
  camera.position.lerpVectors(cameraResetFromPosition, initialCameraPosition, resetT);
  controls.target.lerpVectors(cameraResetFromTarget, initialControlsTarget, resetT);
  camera.lookAt(controls.target);
  if (resetT >= 1.0) cameraResetActive = false;
}
  if (showColony && !cameraResetActive) {
  var prevCameraY = camera.position.y;
  camera.position.y = Math.max(camera.position.y, groundY + cameraFloorOffset);
  if (camera.position.y !== prevCameraY) camera.lookAt(controls.target);
}
  renderer.render(scene, camera);
}

window.addEventListener('resize', function() {
  camera.aspect = window.innerWidth / window.innerHeight;
  camera.updateProjectionMatrix();
  renderer.setSize(window.innerWidth, window.innerHeight);
});

animate();
</script>
</body>
</html>`)

	return sb.String()
}
