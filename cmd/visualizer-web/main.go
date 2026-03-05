package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"lem-in/internal/format"
)

// jsonRoom is the JSON representation of a room for the HTML template.
type jsonRoom struct {
	Name    string  `json:"name"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Z       float64 `json:"z"`
	IsStart bool    `json:"isStart"`
	IsEnd   bool    `json:"isEnd"`
}

// jsonLink is the JSON representation of a tunnel for the HTML template.
type jsonLink struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// jsonMovement is the JSON representation of a single ant move.
type jsonMovement struct {
	AntID    int    `json:"antId"`
	RoomName string `json:"room"`
}

// jsonData is the top-level JSON embedded into the HTML.
type jsonData struct {
	AntCount  int              `json:"antCount"`
	Rooms     []jsonRoom       `json:"rooms"`
	Links     []jsonLink       `json:"links"`
	StartName string           `json:"startName"`
	EndName   string           `json:"endName"`
	Turns     [][]jsonMovement `json:"turns"`
	Error     string           `json:"error,omitempty"`
}

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

	html := buildHTML(string(jsonBytes))
	fmt.Print(html)
}

// bfsDepth computes shortest distance from the start room to every other room.
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

	// Rooms unreachable from start get depth 0
	for _, r := range rooms {
		if _, ok := depth[r.Name]; !ok {
			depth[r.Name] = 0
		}
	}

	return depth
}

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

	// Convert rooms: input X -> Three.js X, input Y -> Three.js Z, BFS depth -> Y (height)
	scale := 4.0
	for _, r := range parsed.Rooms {
		jr := jsonRoom{
			Name:    r.Name,
			X:       float64(r.X) * scale,
			Y:       float64(depths[r.Name]) * scale * 1.5,
			Z:       float64(r.Y) * scale,
			IsStart: r.IsStart,
			IsEnd:   r.IsEnd,
		}
		data.Rooms = append(data.Rooms, jr)
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

func buildHTML(jsonStr string) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>lem-in 3D Visualizer</title>
<style>
*{margin:0;padding:0;box-sizing:border-box;}
html,body{width:100%;height:100%;overflow:hidden;background:#0a0a0f;font-family:'Segoe UI',Tahoma,Geneva,Verdana,sans-serif;color:#c8c8d0;}
canvas{display:block;}
#error-overlay{position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(10,10,15,0.95);
  display:flex;align-items:center;justify-content:center;z-index:1000;flex-direction:column;}
#error-overlay h1{color:#ff4444;font-size:2rem;margin-bottom:1rem;text-shadow:0 0 20px rgba(255,68,68,0.5);}
#error-overlay p{color:#aa8888;font-size:1.1rem;max-width:600px;text-align:center;line-height:1.6;}
#ui-overlay{position:fixed;top:0;left:0;width:100%;height:100%;pointer-events:none;z-index:100;}
#controls{position:absolute;bottom:20px;left:50%;transform:translateX(-50%);
  background:rgba(15,15,25,0.85);border:1px solid rgba(100,100,140,0.3);
  border-radius:12px;padding:14px 24px;display:flex;align-items:center;gap:16px;
  pointer-events:all;backdrop-filter:blur(10px);box-shadow:0 4px 30px rgba(0,0,0,0.5);}
#controls button{background:rgba(80,80,120,0.4);border:1px solid rgba(120,120,160,0.4);
  color:#c8c8d0;padding:8px 16px;border-radius:8px;cursor:pointer;font-size:0.9rem;
  transition:all 0.2s;}
#controls button:hover{background:rgba(100,100,150,0.5);border-color:rgba(150,150,200,0.5);}
#controls button.active{background:rgba(60,120,200,0.4);border-color:rgba(80,140,220,0.5);}
#speed-container{display:flex;align-items:center;gap:8px;}
#speed-slider{-webkit-appearance:none;appearance:none;width:100px;height:4px;
  background:rgba(100,100,140,0.4);border-radius:2px;outline:none;}
#speed-slider::-webkit-slider-thumb{-webkit-appearance:none;appearance:none;width:14px;height:14px;
  background:#6688cc;border-radius:50%;cursor:pointer;transition:background 0.2s;}
#speed-slider::-webkit-slider-thumb:hover{background:#88aaee;}
#turn-display{position:absolute;top:20px;right:20px;
  background:rgba(15,15,25,0.85);border:1px solid rgba(100,100,140,0.3);
  border-radius:10px;padding:12px 20px;pointer-events:all;
  backdrop-filter:blur(10px);box-shadow:0 4px 30px rgba(0,0,0,0.5);}
#turn-display .label{font-size:0.75rem;color:#888;text-transform:uppercase;letter-spacing:1px;}
#turn-display .value{font-size:1.4rem;font-weight:600;color:#aabbdd;}
#info-panel{position:absolute;top:20px;left:20px;
  background:rgba(15,15,25,0.85);border:1px solid rgba(100,100,140,0.3);
  border-radius:10px;padding:12px 20px;pointer-events:all;
  backdrop-filter:blur(10px);box-shadow:0 4px 30px rgba(0,0,0,0.5);}
#info-panel .title{font-size:1rem;font-weight:600;color:#aabbdd;margin-bottom:4px;}
#info-panel .detail{font-size:0.8rem;color:#888;line-height:1.5;}
#hover-label{position:absolute;padding:6px 12px;background:rgba(15,15,25,0.9);
  border:1px solid rgba(100,100,140,0.4);border-radius:6px;font-size:0.85rem;
  pointer-events:none;display:none;white-space:nowrap;
  box-shadow:0 2px 15px rgba(0,0,0,0.4);}
#minimap{position:absolute;bottom:80px;right:20px;width:180px;height:180px;
  background:rgba(15,15,25,0.85);border:1px solid rgba(100,100,140,0.3);
  border-radius:10px;pointer-events:all;overflow:hidden;
  backdrop-filter:blur(10px);box-shadow:0 4px 30px rgba(0,0,0,0.5);}
#minimap canvas{width:100%;height:100%;}
</style>
</head>
<body>
<div id="ui-overlay">
  <div id="turn-display">
    <div class="label">Turn</div>
    <div class="value" id="turn-value">0 / 0</div>
  </div>
  <div id="info-panel">
    <div class="title">lem-in 3D</div>
    <div class="detail" id="info-detail">Loading...</div>
  </div>
  <div id="controls">
    <button id="btn-restart" title="Restart">&#8634;</button>
    <button id="btn-prev" title="Previous Turn">&#9664;</button>
    <button id="btn-play" class="active" title="Play/Pause">&#9654;</button>
    <button id="btn-next" title="Next Turn">&#9654;</button>
    <div id="speed-container">
      <span style="font-size:0.8rem;color:#888;">Speed</span>
      <input type="range" id="speed-slider" min="0.2" max="4" step="0.1" value="1">
      <span id="speed-value" style="font-size:0.8rem;min-width:30px;">1.0x</span>
    </div>
  </div>
  <div id="hover-label"></div>
  <div id="minimap"><canvas id="minimap-canvas" width="360" height="360"></canvas></div>
</div>

<script>
const SIM_DATA = `)
	sb.WriteString(jsonStr)
	sb.WriteString(`;
</script>

<script type="importmap">
{
  "imports": {
    "three": "https://cdn.jsdelivr.net/npm/three@0.160.0/build/three.module.js",
    "three/addons/": "https://cdn.jsdelivr.net/npm/three@0.160.0/examples/jsm/"
  }
}
</script>

<script type="module">
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

// ---------- ERROR HANDLING ----------
if (SIM_DATA.error) {
  document.getElementById('ui-overlay').style.display = 'none';
  const overlay = document.createElement('div');
  overlay.id = 'error-overlay';
  overlay.innerHTML = '<h1>Error</h1><p>' + SIM_DATA.error.replace(/</g,'&lt;') + '</p>';
  document.body.appendChild(overlay);
  throw new Error('Input error');
}

// ---------- SCENE SETUP ----------
const scene = new THREE.Scene();
scene.background = new THREE.Color(0x0a0a12);
scene.fog = new THREE.FogExp2(0x0a0a12, 0.012);

const camera = new THREE.PerspectiveCamera(60, window.innerWidth / window.innerHeight, 0.1, 500);
const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: false });
renderer.setSize(window.innerWidth, window.innerHeight);
renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
renderer.shadowMap.enabled = true;
renderer.shadowMap.type = THREE.PCFSoftShadowMap;
renderer.toneMapping = THREE.ACESFilmicToneMapping;
renderer.toneMappingExposure = 0.8;
document.body.prepend(renderer.domElement);

const controls = new OrbitControls(camera, renderer.domElement);
controls.enableDamping = true;
controls.dampingFactor = 0.08;
controls.rotateSpeed = 0.6;
controls.zoomSpeed = 1.0;
controls.panSpeed = 0.5;
controls.minDistance = 3;
controls.maxDistance = 200;

// ---------- LIGHTING ----------
const ambientLight = new THREE.AmbientLight(0x1a1a2e, 0.6);
scene.add(ambientLight);

const hemiLight = new THREE.HemisphereLight(0x2244aa, 0x332211, 0.3);
scene.add(hemiLight);

// ---------- BUILD ROOM LOOKUP ----------
const roomMap = {};
const rooms = SIM_DATA.rooms || [];
const links = SIM_DATA.links || [];
const turns = SIM_DATA.turns || [];

let centerX = 0, centerY = 0, centerZ = 0;
rooms.forEach(r => { centerX += r.x; centerY += r.y; centerZ += r.z; });
if (rooms.length > 0) {
  centerX /= rooms.length; centerY /= rooms.length; centerZ /= rooms.length;
}

// ---------- ROOM SPHERES ----------
const roomMeshes = [];
const roomGroup = new THREE.Group();
scene.add(roomGroup);

rooms.forEach(r => {
  const px = r.x - centerX;
  const py = r.y - centerY;
  const pz = r.z - centerZ;

  let color, emissive, radius;
  if (r.isStart) {
    color = 0x22cc66; emissive = 0x116633; radius = 0.8;
  } else if (r.isEnd) {
    color = 0xcc3333; emissive = 0x661919; radius = 0.8;
  } else {
    color = 0x887766; emissive = 0x332211; radius = 0.5;
  }

  const geom = new THREE.SphereGeometry(radius, 32, 24);
  const mat = new THREE.MeshStandardMaterial({
    color: color, emissive: emissive, emissiveIntensity: 0.6,
    roughness: 0.4, metalness: 0.3
  });
  const mesh = new THREE.Mesh(geom, mat);
  mesh.position.set(px, py, pz);
  mesh.userData = { roomName: r.name, isStart: r.isStart, isEnd: r.isEnd, baseColor: color, baseEmissive: emissive };
  roomGroup.add(mesh);
  roomMeshes.push(mesh);

  // Point light at each room for cave glow
  const intensity = (r.isStart || r.isEnd) ? 1.5 : 0.4;
  const lightColor = r.isStart ? 0x44ff88 : (r.isEnd ? 0xff4444 : 0xaa9977);
  const pointLight = new THREE.PointLight(lightColor, intensity, 15, 2);
  pointLight.position.copy(mesh.position);
  scene.add(pointLight);

  roomMap[r.name] = { mesh, pos: new THREE.Vector3(px, py, pz) };
});

// ---------- TUNNEL LINES ----------
const tunnelGroup = new THREE.Group();
scene.add(tunnelGroup);

const linkObjects = [];
links.forEach(l => {
  const fromRoom = roomMap[l.from];
  const toRoom = roomMap[l.to];
  if (!fromRoom || !toRoom) return;

  // Create a tube-like line with LineSegments
  const points = [];
  const segCount = 20;
  for (let i = 0; i <= segCount; i++) {
    const t = i / segCount;
    const p = new THREE.Vector3().lerpVectors(fromRoom.pos, toRoom.pos, t);
    // Slight sag for visual effect
    const sag = Math.sin(t * Math.PI) * 0.3;
    p.y -= sag;
    points.push(p);
  }

  const curve = new THREE.CatmullRomCurve3(points);
  const tubeGeom = new THREE.TubeGeometry(curve, 16, 0.06, 8, false);
  const tubeMat = new THREE.MeshStandardMaterial({
    color: 0x554433, emissive: 0x221100, emissiveIntensity: 0.2,
    roughness: 0.7, metalness: 0.2, transparent: true, opacity: 0.7
  });
  const tube = new THREE.Mesh(tubeGeom, tubeMat);
  tube.userData = { from: l.from, to: l.to };
  tunnelGroup.add(tube);
  linkObjects.push(tube);
});

// ---------- CAVE PARTICLES (dust / atmosphere) ----------
const dustCount = 600;
const dustGeom = new THREE.BufferGeometry();
const dustPositions = new Float32Array(dustCount * 3);
const dustSizes = new Float32Array(dustCount);
const spread = 60;
for (let i = 0; i < dustCount; i++) {
  dustPositions[i * 3] = (Math.random() - 0.5) * spread;
  dustPositions[i * 3 + 1] = (Math.random() - 0.5) * spread;
  dustPositions[i * 3 + 2] = (Math.random() - 0.5) * spread;
  dustSizes[i] = Math.random() * 2 + 0.5;
}
dustGeom.setAttribute('position', new THREE.BufferAttribute(dustPositions, 3));
dustGeom.setAttribute('size', new THREE.BufferAttribute(dustSizes, 1));
const dustMat = new THREE.PointsMaterial({
  color: 0x665544, size: 0.15, transparent: true, opacity: 0.25,
  sizeAttenuation: true, blending: THREE.AdditiveBlending, depthWrite: false
});
const dustSystem = new THREE.Points(dustGeom, dustMat);
scene.add(dustSystem);

// ---------- CAMERA POSITION ----------
let maxDist = 0;
rooms.forEach(r => {
  const d = Math.sqrt(
    (r.x - centerX) ** 2 + (r.y - centerY) ** 2 + (r.z - centerZ) ** 2
  );
  if (d > maxDist) maxDist = d;
});
const camDist = Math.max(maxDist * 2.5, 15);
camera.position.set(camDist * 0.6, camDist * 0.8, camDist * 0.6);
camera.lookAt(0, 0, 0);
controls.target.set(0, 0, 0);

// ---------- ANT MANAGEMENT ----------
const antMeshes = {};   // antId -> mesh
const antTrails = {};   // antId -> trail points array
const antPrevRoom = {}; // antId -> previous room name (for path computation)

// Track which room each ant is at before the simulation starts
// Before turn 1, all ants are at the start room.
const antCurrentRoom = {}; // antId -> room name

function getAntColor(antId) {
  const hue = (antId * 137.508) % 360;
  return new THREE.Color().setHSL(hue / 360, 0.7, 0.55);
}

function getOrCreateAnt(antId) {
  if (antMeshes[antId]) return antMeshes[antId];
  const color = getAntColor(antId);
  const geom = new THREE.SphereGeometry(0.25, 16, 12);
  const mat = new THREE.MeshStandardMaterial({
    color: color, emissive: color, emissiveIntensity: 0.8,
    roughness: 0.2, metalness: 0.5
  });
  const mesh = new THREE.Mesh(geom, mat);
  mesh.visible = false;
  scene.add(mesh);
  antMeshes[antId] = mesh;

  // Glow
  const glowGeom = new THREE.SphereGeometry(0.4, 16, 12);
  const glowMat = new THREE.MeshBasicMaterial({
    color: color, transparent: true, opacity: 0.2, blending: THREE.AdditiveBlending
  });
  const glow = new THREE.Mesh(glowGeom, glowMat);
  mesh.add(glow);

  // Point light on ant
  const antLight = new THREE.PointLight(color.getHex(), 0.6, 6, 2);
  mesh.add(antLight);

  antTrails[antId] = [];
  antCurrentRoom[antId] = SIM_DATA.startName;

  return mesh;
}

// ---------- TRAIL SYSTEM ----------
const trailParticles = [];
const maxTrailParticles = 5000;

function addTrailPoint(position, color) {
  const geom = new THREE.SphereGeometry(0.08, 4, 4);
  const mat = new THREE.MeshBasicMaterial({
    color: color, transparent: true, opacity: 0.6, blending: THREE.AdditiveBlending
  });
  const p = new THREE.Mesh(geom, mat);
  p.position.copy(position);
  p.userData.life = 1.0;
  scene.add(p);
  trailParticles.push(p);

  // Remove old particles
  while (trailParticles.length > maxTrailParticles) {
    const old = trailParticles.shift();
    scene.remove(old);
    old.geometry.dispose();
    old.material.dispose();
  }
}

function updateTrails(dt) {
  const fadeSpeed = 0.4;
  for (let i = trailParticles.length - 1; i >= 0; i--) {
    const p = trailParticles[i];
    p.userData.life -= dt * fadeSpeed;
    p.material.opacity = Math.max(0, p.userData.life * 0.5);
    p.scale.setScalar(Math.max(0.1, p.userData.life));
    if (p.userData.life <= 0) {
      scene.remove(p);
      p.geometry.dispose();
      p.material.dispose();
      trailParticles.splice(i, 1);
    }
  }
}

// ---------- ANIMATION STATE ----------
let currentTurn = 0;
let turnProgress = 0;  // 0..1 within current turn
let isPlaying = true;
let speed = 1.0;
let animComplete = turns.length === 0;

// Pre-compute animation paths for each turn
// For each turn, for each ant movement, we need: fromPos, toPos, antId
// We need to track where each ant currently is to determine the "from" position.
function buildTurnAnimations() {
  const anims = [];
  const antPos = {};  // track positions through turns

  for (let t = 0; t < turns.length; t++) {
    const turnAnims = [];
    for (const m of turns[t]) {
      const fromName = antPos[m.antId] || SIM_DATA.startName;
      const toName = m.room;
      const fromRoom = roomMap[fromName];
      const toRoom = roomMap[toName];
      if (fromRoom && toRoom) {
        turnAnims.push({
          antId: m.antId,
          fromPos: fromRoom.pos.clone(),
          toPos: toRoom.pos.clone(),
          fromName: fromName,
          toName: toName
        });
      }
      antPos[m.antId] = toName;
    }
    anims.push(turnAnims);
  }
  return anims;
}

const turnAnimations = buildTurnAnimations();

// ---------- CUBIC BEZIER INTERPOLATION ----------
function cubicBezier(p0, p1, p2, p3, t) {
  const u = 1 - t;
  return new THREE.Vector3(
    u*u*u*p0.x + 3*u*u*t*p1.x + 3*u*t*t*p2.x + t*t*t*p3.x,
    u*u*u*p0.y + 3*u*u*t*p1.y + 3*u*t*t*p2.y + t*t*t*p3.y,
    u*u*u*p0.z + 3*u*u*t*p1.z + 3*u*t*t*p2.z + t*t*t*p3.z
  );
}

function getAnimPosition(from, to, t) {
  // Cubic Bezier: lift ants above the tunnel with an arc
  const mid = new THREE.Vector3().lerpVectors(from, to, 0.5);
  const dist = from.distanceTo(to);
  const lift = dist * 0.4;
  const cp1 = new THREE.Vector3().lerpVectors(from, to, 0.25);
  cp1.y += lift;
  const cp2 = new THREE.Vector3().lerpVectors(from, to, 0.75);
  cp2.y += lift;
  // Ease in-out
  const et = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
  return cubicBezier(from, cp1, cp2, to, et);
}

// ---------- RAYCASTING (HOVER) ----------
const raycaster = new THREE.Raycaster();
const mouse = new THREE.Vector2();
let hoveredRoom = null;
const hoverLabel = document.getElementById('hover-label');

renderer.domElement.addEventListener('mousemove', (e) => {
  mouse.x = (e.clientX / window.innerWidth) * 2 - 1;
  mouse.y = -(e.clientY / window.innerHeight) * 2 + 1;

  raycaster.setFromCamera(mouse, camera);
  const intersects = raycaster.intersectObjects(roomMeshes);

  if (intersects.length > 0) {
    const hit = intersects[0].object;
    const rName = hit.userData.roomName;
    if (hoveredRoom !== rName) {
      resetHighlights();
      hoveredRoom = rName;
      highlightRoom(rName);
    }
    hoverLabel.style.display = 'block';
    hoverLabel.style.left = (e.clientX + 15) + 'px';
    hoverLabel.style.top = (e.clientY - 10) + 'px';
    let labelText = rName;
    if (hit.userData.isStart) labelText += ' (START)';
    if (hit.userData.isEnd) labelText += ' (END)';
    hoverLabel.textContent = labelText;
  } else {
    if (hoveredRoom) {
      resetHighlights();
      hoveredRoom = null;
    }
    hoverLabel.style.display = 'none';
  }
});

function highlightRoom(roomName) {
  // Highlight the room
  roomMeshes.forEach(m => {
    if (m.userData.roomName === roomName) {
      m.material.emissiveIntensity = 1.5;
      m.scale.setScalar(1.3);
    }
  });
  // Highlight connected tunnels
  linkObjects.forEach(tube => {
    if (tube.userData.from === roomName || tube.userData.to === roomName) {
      tube.material.emissiveIntensity = 0.8;
      tube.material.opacity = 1.0;
      tube.material.color.set(0xaa8855);
    }
  });
}

function resetHighlights() {
  roomMeshes.forEach(m => {
    m.material.emissiveIntensity = 0.6;
    m.scale.setScalar(1.0);
  });
  linkObjects.forEach(tube => {
    tube.material.emissiveIntensity = 0.2;
    tube.material.opacity = 0.7;
    tube.material.color.set(0x554433);
  });
}

// ---------- UI CONTROLS ----------
const btnPlay = document.getElementById('btn-play');
const btnPrev = document.getElementById('btn-prev');
const btnNext = document.getElementById('btn-next');
const btnRestart = document.getElementById('btn-restart');
const speedSlider = document.getElementById('speed-slider');
const speedValue = document.getElementById('speed-value');
const turnValue = document.getElementById('turn-value');
const infoDetail = document.getElementById('info-detail');

infoDetail.textContent = SIM_DATA.antCount + ' ants | ' + rooms.length + ' rooms | ' + links.length + ' tunnels | ' + turns.length + ' turns';

btnPlay.addEventListener('click', () => {
  isPlaying = !isPlaying;
  btnPlay.innerHTML = isPlaying ? '&#10074;&#10074;' : '&#9654;';
  btnPlay.classList.toggle('active', isPlaying);
});

btnRestart.addEventListener('click', () => {
  currentTurn = 0;
  turnProgress = 0;
  animComplete = turns.length === 0;
  hideAllAnts();
  clearTrails();
});

btnPrev.addEventListener('click', () => {
  if (currentTurn > 0) {
    currentTurn--;
    turnProgress = 0;
    animComplete = false;
    rebuildAntsToTurn(currentTurn);
  }
});

btnNext.addEventListener('click', () => {
  if (currentTurn < turns.length - 1) {
    currentTurn++;
    turnProgress = 0;
    animComplete = false;
    rebuildAntsToTurn(currentTurn);
  }
});

speedSlider.addEventListener('input', () => {
  speed = parseFloat(speedSlider.value);
  speedValue.textContent = speed.toFixed(1) + 'x';
});

function hideAllAnts() {
  Object.values(antMeshes).forEach(m => { m.visible = false; });
}

function clearTrails() {
  trailParticles.forEach(p => {
    scene.remove(p);
    p.geometry.dispose();
    p.material.dispose();
  });
  trailParticles.length = 0;
}

function rebuildAntsToTurn(targetTurn) {
  // Reset all ants and replay positions up to targetTurn start
  hideAllAnts();
  clearTrails();
  const antPos = {};
  for (let t = 0; t < targetTurn; t++) {
    for (const m of turns[t]) {
      antPos[m.antId] = m.room;
    }
  }
  // Place ants at their positions
  for (const [idStr, roomName] of Object.entries(antPos)) {
    const id = parseInt(idStr);
    const mesh = getOrCreateAnt(id);
    const room = roomMap[roomName];
    if (room && roomName !== SIM_DATA.endName) {
      mesh.position.copy(room.pos);
      mesh.visible = true;
    }
  }
}

// ---------- MINIMAP ----------
const minimapCanvas = document.getElementById('minimap-canvas');
const mCtx = minimapCanvas.getContext('2d');

function drawMinimap() {
  mCtx.clearRect(0, 0, 360, 360);
  mCtx.fillStyle = 'rgba(10,10,20,0.9)';
  mCtx.fillRect(0, 0, 360, 360);

  if (rooms.length === 0) return;

  // Find bounds (use x, z for top-down)
  let minX = Infinity, maxX = -Infinity, minZ = Infinity, maxZ = -Infinity;
  rooms.forEach(r => {
    const px = r.x - centerX;
    const pz = r.z - centerZ;
    if (px < minX) minX = px;
    if (px > maxX) maxX = px;
    if (pz < minZ) minZ = pz;
    if (pz > maxZ) maxZ = pz;
  });

  const rangeX = maxX - minX || 1;
  const rangeZ = maxZ - minZ || 1;
  const pad = 30;
  const scaleX = (360 - pad * 2) / rangeX;
  const scaleZ = (360 - pad * 2) / rangeZ;
  const sc = Math.min(scaleX, scaleZ);

  function toMinimap(x, z) {
    return {
      mx: pad + (x - minX) * sc + ((360 - pad * 2) - rangeX * sc) / 2,
      my: pad + (z - minZ) * sc + ((360 - pad * 2) - rangeZ * sc) / 2
    };
  }

  // Draw links
  mCtx.strokeStyle = 'rgba(85,68,51,0.5)';
  mCtx.lineWidth = 1;
  links.forEach(l => {
    const fr = roomMap[l.from];
    const tr = roomMap[l.to];
    if (!fr || !tr) return;
    const a = toMinimap(fr.pos.x, fr.pos.z);
    const b = toMinimap(tr.pos.x, tr.pos.z);
    mCtx.beginPath();
    mCtx.moveTo(a.mx, a.my);
    mCtx.lineTo(b.mx, b.my);
    mCtx.stroke();
  });

  // Draw rooms
  rooms.forEach(r => {
    const px = r.x - centerX;
    const pz = r.z - centerZ;
    const { mx, my } = toMinimap(px, pz);
    mCtx.beginPath();
    mCtx.arc(mx, my, r.isStart || r.isEnd ? 5 : 3, 0, Math.PI * 2);
    if (r.isStart) mCtx.fillStyle = '#22cc66';
    else if (r.isEnd) mCtx.fillStyle = '#cc3333';
    else mCtx.fillStyle = '#887766';
    mCtx.fill();
  });

  // Draw ants on minimap
  Object.entries(antMeshes).forEach(([idStr, mesh]) => {
    if (!mesh.visible) return;
    const { mx, my } = toMinimap(mesh.position.x, mesh.position.z);
    mCtx.beginPath();
    mCtx.arc(mx, my, 3, 0, Math.PI * 2);
    const col = getAntColor(parseInt(idStr));
    mCtx.fillStyle = '#' + col.getHexString();
    mCtx.fill();
  });
}

// ---------- MAIN ANIMATION LOOP ----------
const clock = new THREE.Clock();
let trailTimer = 0;

function animate() {
  requestAnimationFrame(animate);
  const dt = clock.getDelta();

  // Dust animation
  const positions = dustGeom.attributes.position;
  for (let i = 0; i < dustCount; i++) {
    positions.array[i * 3 + 1] += Math.sin(Date.now() * 0.001 + i) * 0.002;
    positions.array[i * 3] += Math.cos(Date.now() * 0.0007 + i * 0.5) * 0.001;
  }
  positions.needsUpdate = true;

  // Animation logic
  if (isPlaying && !animComplete && turns.length > 0 && currentTurn < turns.length) {
    turnProgress += dt * speed * 0.8;  // ~1.25 seconds per turn at 1x

    if (turnProgress >= 1.0) {
      // Snap ants to final positions for this turn
      snapAntsToTurnEnd(currentTurn);
      currentTurn++;
      turnProgress = 0;

      if (currentTurn >= turns.length) {
        animComplete = true;
        // Hide ants that reached the end
        hideFinishedAnts();
      }
    } else {
      // Interpolate ant positions
      animateAntsInTurn(currentTurn, turnProgress);
    }
  }

  // Trail fade
  updateTrails(dt);

  // Trail spawning
  trailTimer += dt;
  if (trailTimer > 0.05) {
    trailTimer = 0;
    Object.entries(antMeshes).forEach(([idStr, mesh]) => {
      if (mesh.visible) {
        addTrailPoint(mesh.position.clone(), getAntColor(parseInt(idStr)));
      }
    });
  }

  // Update turn display
  const displayTurn = animComplete ? turns.length : currentTurn + (turns.length > 0 ? 1 : 0);
  turnValue.textContent = displayTurn + ' / ' + turns.length;

  controls.update();
  renderer.render(scene, camera);

  // Minimap at reduced framerate
  if (Math.floor(Date.now() / 100) % 2 === 0) drawMinimap();
}

function animateAntsInTurn(turnIdx, progress) {
  if (turnIdx >= turnAnimations.length) return;
  const anims = turnAnimations[turnIdx];

  for (const anim of anims) {
    const mesh = getOrCreateAnt(anim.antId);
    mesh.visible = true;
    const pos = getAnimPosition(anim.fromPos, anim.toPos, progress);
    mesh.position.copy(pos);
  }
}

function snapAntsToTurnEnd(turnIdx) {
  if (turnIdx >= turnAnimations.length) return;
  const anims = turnAnimations[turnIdx];

  for (const anim of anims) {
    const mesh = getOrCreateAnt(anim.antId);
    mesh.position.copy(anim.toPos);
    mesh.visible = true;
  }
}

function hideFinishedAnts() {
  // Build final positions
  const finalRoom = {};
  for (let t = 0; t < turns.length; t++) {
    for (const m of turns[t]) {
      finalRoom[m.antId] = m.room;
    }
  }
  Object.entries(finalRoom).forEach(([idStr, roomName]) => {
    if (roomName === SIM_DATA.endName) {
      const mesh = antMeshes[parseInt(idStr)];
      if (mesh) mesh.visible = false;
    }
  });
}

// ---------- RESIZE ----------
window.addEventListener('resize', () => {
  camera.aspect = window.innerWidth / window.innerHeight;
  camera.updateProjectionMatrix();
  renderer.setSize(window.innerWidth, window.innerHeight);
});

// Start
animate();
</script>
</body>
</html>`)

	return sb.String()
}
