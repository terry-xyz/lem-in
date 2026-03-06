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
html,body{width:100%;height:100%;overflow:hidden;background:#000;
  font-family:'Segoe UI',system-ui,-apple-system,sans-serif;color:#ccc;}
canvas{display:block;}
#fade-in{position:fixed;top:0;left:0;width:100%;height:100%;background:#000;
  z-index:9999;pointer-events:none;transition:opacity 1.2s ease;opacity:1;}
#error-overlay{position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,0.95);
  display:flex;align-items:center;justify-content:center;z-index:1000;flex-direction:column;}
#error-overlay h1{color:#ff4f75;font-size:2rem;margin-bottom:1rem;
  text-shadow:0 0 30px rgba(255,79,117,0.5);}
#error-overlay p{color:#aa8888;font-size:1.1rem;max-width:600px;text-align:center;line-height:1.6;}
#vignette{position:fixed;top:0;left:0;width:100%;height:100%;pointer-events:none;z-index:98;
  background:radial-gradient(ellipse at center, transparent 50%, rgba(0,0,0,0.55) 100%);}
#ui-overlay{position:fixed;top:0;left:0;width:100%;height:100%;pointer-events:none;z-index:100;}
#controls{position:absolute;bottom:20px;left:50%;transform:translateX(-50%);
  background:rgba(10,10,20,0.55);backdrop-filter:blur(20px);-webkit-backdrop-filter:blur(20px);
  border:1px solid rgba(68,136,204,0.1);
  border-radius:16px;padding:14px 28px;display:flex;align-items:center;gap:16px;
  pointer-events:all;box-shadow:0 8px 32px rgba(0,0,0,0.4);}
#controls button{background:rgba(68,136,204,0.08);border:1px solid rgba(68,136,204,0.12);
  color:#aaa;padding:8px 16px;border-radius:8px;cursor:pointer;font-size:0.9rem;
  transition:all 0.3s ease;}
#controls button:hover{background:rgba(68,136,204,0.18);border-color:rgba(68,136,204,0.3);
  color:#ddd;box-shadow:0 0 15px rgba(68,136,204,0.08);}
#controls button.active{background:rgba(68,136,204,0.22);border-color:rgba(68,136,204,0.35);
  color:#fff;box-shadow:0 0 20px rgba(68,136,204,0.12);}
#speed-container{display:flex;align-items:center;gap:8px;}
#speed-slider{-webkit-appearance:none;appearance:none;width:100px;height:3px;
  background:rgba(68,136,204,0.15);border-radius:2px;outline:none;}
#speed-slider::-webkit-slider-thumb{-webkit-appearance:none;appearance:none;width:14px;height:14px;
  background:#4488cc;border-radius:50%;cursor:pointer;transition:all 0.2s;
  box-shadow:0 0 10px rgba(68,136,204,0.4);}
#speed-slider::-webkit-slider-thumb:hover{background:#66aaee;
  box-shadow:0 0 15px rgba(68,136,204,0.6);}
#timeline-container{display:flex;align-items:center;gap:8px;flex:1;}
#timeline-slider{-webkit-appearance:none;appearance:none;width:100%;height:3px;
  background:rgba(68,136,204,0.15);border-radius:2px;outline:none;}
#timeline-slider::-webkit-slider-thumb{-webkit-appearance:none;appearance:none;width:14px;height:14px;
  background:#4488cc;border-radius:50%;cursor:pointer;transition:all 0.2s;
  box-shadow:0 0 10px rgba(68,136,204,0.4);}
#timeline-slider::-webkit-slider-thumb:hover{background:#66aaee;
  box-shadow:0 0 15px rgba(68,136,204,0.6);}
#turn-display{position:absolute;top:20px;right:20px;
  background:rgba(10,10,20,0.55);backdrop-filter:blur(20px);-webkit-backdrop-filter:blur(20px);
  border:1px solid rgba(68,136,204,0.1);
  border-radius:12px;padding:12px 20px;pointer-events:all;
  box-shadow:0 8px 32px rgba(0,0,0,0.4);}
#turn-display .label{font-size:0.7rem;color:#555;text-transform:uppercase;letter-spacing:2px;}
#turn-display .value{font-size:1.4rem;font-weight:600;color:#4488cc;
  text-shadow:0 0 15px rgba(68,136,204,0.3);}
#info-panel{position:absolute;top:20px;left:20px;
  background:rgba(10,10,20,0.55);backdrop-filter:blur(20px);-webkit-backdrop-filter:blur(20px);
  border:1px solid rgba(68,136,204,0.1);
  border-radius:12px;padding:12px 20px;pointer-events:all;
  box-shadow:0 8px 32px rgba(0,0,0,0.4);}
#info-panel .title{font-size:1rem;font-weight:600;color:#4488cc;margin-bottom:4px;
  text-shadow:0 0 15px rgba(68,136,204,0.3);}
#info-panel .detail{font-size:0.8rem;color:#555;line-height:1.5;}
#hover-label{position:absolute;padding:8px 14px;
  background:rgba(10,10,20,0.7);backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);
  border:1px solid rgba(68,136,204,0.12);border-radius:8px;font-size:0.8rem;
  pointer-events:none;display:none;white-space:pre-line;line-height:1.5;
  box-shadow:0 4px 15px rgba(0,0,0,0.3);}
</style>
</head>
<body>
<div id="fade-in"></div>
<div id="vignette"></div>
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
    <div id="timeline-container">
      <input type="range" id="timeline-slider" min="0" max="0" step="1" value="0">
    </div>
    <div id="speed-container">
      <span style="font-size:0.8rem;color:#555;">Speed</span>
      <input type="range" id="speed-slider" min="0.2" max="4" step="0.1" value="1">
      <span id="speed-value" style="font-size:0.8rem;min-width:30px;">1.0x</span>
    </div>
  </div>
  <div id="hover-label"></div>
</div>

<script>
const SIM_DATA = `)
	sb.WriteString(jsonStr)
	sb.WriteString(`;
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
import { EffectComposer } from 'three/addons/postprocessing/EffectComposer.js';
import { RenderPass } from 'three/addons/postprocessing/RenderPass.js';
import { UnrealBloomPass } from 'three/addons/postprocessing/UnrealBloomPass.js';
import { OutputPass } from 'three/addons/postprocessing/OutputPass.js';

// ---------- ERROR HANDLING ----------
if (SIM_DATA.error) {
  document.getElementById('ui-overlay').style.display = 'none';
  document.getElementById('fade-in').style.display = 'none';
  const overlay = document.createElement('div');
  overlay.id = 'error-overlay';
  overlay.innerHTML = '<h1>Error</h1><p>' + SIM_DATA.error.replace(/</g,'&lt;') + '</p>';
  document.body.appendChild(overlay);
  throw new Error('Input error');
}

// ---------- SCENE SETUP ----------
const scene = new THREE.Scene();
scene.background = new THREE.Color(0x020208);

const camera = new THREE.PerspectiveCamera(60, window.innerWidth / window.innerHeight, 0.1, 500);
const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: false });
renderer.setSize(window.innerWidth, window.innerHeight);
renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
renderer.toneMapping = THREE.ACESFilmicToneMapping;
renderer.toneMappingExposure = 1.0;
document.body.prepend(renderer.domElement);

// ---------- POST-PROCESSING (BLOOM) ----------
const composer = new EffectComposer(renderer);
composer.addPass(new RenderPass(scene, camera));
const bloomPass = new UnrealBloomPass(
  new THREE.Vector2(window.innerWidth, window.innerHeight),
  1.2, 0.5, 0.2
);
composer.addPass(bloomPass);
composer.addPass(new OutputPass());

const controls = new OrbitControls(camera, renderer.domElement);
controls.enableDamping = true;
controls.dampingFactor = 0.15;
controls.rotateSpeed = 0.6;
controls.zoomSpeed = 1.0;
controls.panSpeed = 0.5;
controls.minDistance = 3;
controls.maxDistance = 200;
controls.enabled = false;
controls.autoRotate = true;
controls.autoRotateSpeed = 0.3;

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

// ---------- ROOM SPHERES (MeshBasicMaterial + Glow + Orbital Rings) ----------
const roomMeshes = [];
const roomGroup = new THREE.Group();
scene.add(roomGroup);
const orbitalRings = [];

rooms.forEach(r => {
  const px = r.x - centerX;
  const py = r.y - centerY;
  const pz = r.z - centerZ;

  let color, radius;
  if (r.isStart) {
    color = 0x4ADE80; radius = 0.8;
  } else if (r.isEnd) {
    color = 0xff4f75; radius = 0.8;
  } else {
    color = 0x4488cc; radius = 0.5;
  }

  const geom = new THREE.SphereGeometry(radius, 32, 24);
  const mat = new THREE.MeshBasicMaterial({ color: color });
  const mesh = new THREE.Mesh(geom, mat);
  mesh.userData = { roomName: r.name, isStart: r.isStart, isEnd: r.isEnd, baseColor: color };
  // Scatter position for fly-in animation (inspired by Particle Globe implosion)
  const sf = 3 + Math.random() * 4;
  mesh.userData.finalPos = new THREE.Vector3(px, py, pz);
  mesh.userData.scatterPos = new THREE.Vector3(
    px * sf + (Math.random() - 0.5) * 10,
    py * sf + (Math.random() - 0.5) * 10,
    pz * sf + (Math.random() - 0.5) * 10
  );
  mesh.position.copy(mesh.userData.scatterPos);
  roomGroup.add(mesh);
  roomMeshes.push(mesh);

  // Glow shell
  const glowGeom = new THREE.SphereGeometry(radius * 2, 32, 24);
  const glowMat = new THREE.MeshBasicMaterial({
    color: color, transparent: true, opacity: 0.08,
    blending: THREE.AdditiveBlending, depthWrite: false
  });
  mesh.add(new THREE.Mesh(glowGeom, glowMat));

  // Orbital rings for start/end nodes
  if (r.isStart || r.isEnd) {
    const ringGeom = new THREE.TorusGeometry(radius * 1.8, 0.03, 8, 48);
    const ringMat = new THREE.MeshBasicMaterial({
      color: color, transparent: true, opacity: 0.35,
      blending: THREE.AdditiveBlending, depthWrite: false
    });
    const ring1 = new THREE.Mesh(ringGeom, ringMat);
    ring1.rotation.x = Math.PI * 0.5;
    mesh.add(ring1);
    orbitalRings.push(ring1);

    const ring2Geom = new THREE.TorusGeometry(radius * 2.2, 0.025, 8, 48);
    const ring2 = new THREE.Mesh(ring2Geom, ringMat.clone());
    ring2.rotation.x = Math.PI * 0.3;
    ring2.rotation.z = Math.PI * 0.4;
    mesh.add(ring2);
    orbitalRings.push(ring2);
  }

  roomMap[r.name] = { mesh, pos: new THREE.Vector3(px, py, pz) };
});

// ---------- EDGES (Line + LineBasicMaterial) ----------
const tunnelGroup = new THREE.Group();
tunnelGroup.visible = false;
scene.add(tunnelGroup);

const linkObjects = [];
links.forEach(l => {
  const fromRoom = roomMap[l.from];
  const toRoom = roomMap[l.to];
  if (!fromRoom || !toRoom) return;

  const mid = fromRoom.pos.clone().lerp(toRoom.pos, 0.5);
  mid.y += fromRoom.pos.distanceTo(toRoom.pos) * 0.06;
  const curve = new THREE.CatmullRomCurve3([fromRoom.pos.clone(), mid, toRoom.pos.clone()]);
  const tubeGeom = new THREE.TubeGeometry(curve, 12, 0.04, 6, false);
  const mat = new THREE.MeshBasicMaterial({
    color: 0x1a2a3a, transparent: true, opacity: 0.7,
    blending: THREE.AdditiveBlending, depthWrite: false
  });
  const tube = new THREE.Mesh(tubeGeom, mat);
  tube.userData = { from: l.from, to: l.to, baseColor: 0x1a2a3a, baseOpacity: 0.7 };
  tunnelGroup.add(tube);
  linkObjects.push(tube);
});

// ---------- CAMERA POSITION ----------
let maxDist = 0;
rooms.forEach(r => {
  const d = Math.sqrt(
    (r.x - centerX) ** 2 + (r.y - centerY) ** 2 + (r.z - centerZ) ** 2
  );
  if (d > maxDist) maxDist = d;
});
const camDist = Math.max(maxDist * 2.5, 15);
scene.fog = new THREE.FogExp2(0x020208, 0.8 / camDist);
const camEnd = new THREE.Vector3(camDist * 0.6, camDist * 0.8, camDist * 0.6);
const camStart = camEnd.clone().multiplyScalar(2.5);
camera.position.copy(camStart);
camera.lookAt(0, 0, 0);
controls.target.set(0, 0, 0);
let introProgress = 0;

// ---------- AMBIENT PARTICLES (Firefly-style) ----------
const PARTICLE_COUNT = 200;
const pPositions = new Float32Array(PARTICLE_COUNT * 3);
const pVelocities = [];
const pSpread = Math.max(maxDist * 3, 30);

for (let i = 0; i < PARTICLE_COUNT; i++) {
  pPositions[i * 3] = (Math.random() - 0.5) * pSpread;
  pPositions[i * 3 + 1] = (Math.random() - 0.5) * pSpread;
  pPositions[i * 3 + 2] = (Math.random() - 0.5) * pSpread;
  pVelocities.push({
    vx: (Math.random() - 0.5) * 0.3,
    vy: (Math.random() - 0.5) * 0.15,
    vz: (Math.random() - 0.5) * 0.3,
    phase: Math.random() * Math.PI * 2
  });
}

const pGeom = new THREE.BufferGeometry();
pGeom.setAttribute('position', new THREE.BufferAttribute(pPositions, 3));

// Soft circular texture (SPH kernel-inspired radial falloff from Firefly)
const pCanvas = document.createElement('canvas');
pCanvas.width = 32; pCanvas.height = 32;
const pCtx = pCanvas.getContext('2d');
const grad = pCtx.createRadialGradient(16, 16, 0, 16, 16, 16);
grad.addColorStop(0, 'rgba(255,255,255,1)');
grad.addColorStop(0.3, 'rgba(255,255,255,0.6)');
grad.addColorStop(0.7, 'rgba(255,255,255,0.15)');
grad.addColorStop(1, 'rgba(255,255,255,0)');
pCtx.fillStyle = grad;
pCtx.fillRect(0, 0, 32, 32);
const pTex = new THREE.CanvasTexture(pCanvas);

const pMat = new THREE.PointsMaterial({
  color: 0x6699cc, size: 0.5, transparent: true, opacity: 0.6,
  blending: THREE.AdditiveBlending, depthWrite: false, sizeAttenuation: true,
  map: pTex
});
const ambientParticles = new THREE.Points(pGeom, pMat);
scene.add(ambientParticles);

// ---------- ANT MANAGEMENT ----------
const antMeshes = {};
const antCurrentRoom = {};

function getAntColor(antId) {
  const hue = (antId * 137.508) % 360;
  return new THREE.Color().setHSL(hue / 360, 0.7, 0.55);
}

function getOrCreateAnt(antId) {
  if (antMeshes[antId]) return antMeshes[antId];
  const color = getAntColor(antId);
  const geom = new THREE.SphereGeometry(0.25, 16, 12);
  const mat = new THREE.MeshBasicMaterial({ color: color });
  const mesh = new THREE.Mesh(geom, mat);
  mesh.visible = false;
  mesh.userData = { baseColor: color.getHex() };
  scene.add(mesh);
  antMeshes[antId] = mesh;

  // Glow shell
  const glowGeom = new THREE.SphereGeometry(0.5, 16, 12);
  const glowMat = new THREE.MeshBasicMaterial({
    color: color, transparent: true, opacity: 0.2,
    blending: THREE.AdditiveBlending, depthWrite: false
  });
  mesh.add(new THREE.Mesh(glowGeom, glowMat));

  antCurrentRoom[antId] = SIM_DATA.startName;
  return mesh;
}

// ---------- TRAIL SYSTEM (BufferGeometry Ring Buffer) ----------
const antTrails = {};
const TRAIL_LEN = 128;

function getOrCreateTrail(antId) {
  if (antTrails[antId]) return antTrails[antId];

  const positions = new Float32Array(TRAIL_LEN * 3);
  const colors = new Float32Array(TRAIL_LEN * 3);
  const geom = new THREE.BufferGeometry();
  geom.setAttribute('position', new THREE.BufferAttribute(positions, 3));
  geom.setAttribute('color', new THREE.BufferAttribute(colors, 3));
  geom.setDrawRange(0, 0);

  const mat = new THREE.LineBasicMaterial({
    vertexColors: true, transparent: true, opacity: 0.8,
    blending: THREE.AdditiveBlending, depthWrite: false
  });
  const line = new THREE.Line(geom, mat);
  scene.add(line);

  const trail = { line, positions, colors, writeIdx: 0, count: 0 };
  antTrails[antId] = trail;
  return trail;
}

function addTrailSample(antId, position) {
  const trail = getOrCreateTrail(antId);
  const c = getAntColor(antId);
  const i = trail.writeIdx;
  trail.positions[i * 3] = position.x;
  trail.positions[i * 3 + 1] = position.y;
  trail.positions[i * 3 + 2] = position.z;
  trail.colors[i * 3] = c.r;
  trail.colors[i * 3 + 1] = c.g;
  trail.colors[i * 3 + 2] = c.b;
  trail.writeIdx = (trail.writeIdx + 1) % TRAIL_LEN;
  if (trail.count < TRAIL_LEN) trail.count++;
  trail.line.geometry.setDrawRange(0, trail.count);
  trail.line.geometry.attributes.position.needsUpdate = true;
  trail.line.geometry.attributes.color.needsUpdate = true;
}

function fadeTrails(dt) {
  const decay = 1 - dt * 1.5;
  Object.values(antTrails).forEach(trail => {
    const c = trail.colors;
    for (let i = 0; i < trail.count * 3; i++) {
      c[i] *= decay;
      if (c[i] < 0.001) c[i] = 0;
    }
    trail.line.geometry.attributes.color.needsUpdate = true;
  });
}

function clearTrails() {
  Object.values(antTrails).forEach(trail => {
    trail.positions.fill(0);
    trail.colors.fill(0);
    trail.writeIdx = 0;
    trail.count = 0;
    trail.line.geometry.setDrawRange(0, 0);
    trail.line.geometry.attributes.position.needsUpdate = true;
    trail.line.geometry.attributes.color.needsUpdate = true;
  });
}

// ---------- ARRIVAL BURST SYSTEM (Pool-based) ----------
const BURST_SLOTS = 8;
const BURST_PARTICLES = 12;
const bursts = [];
for (let s = 0; s < BURST_SLOTS; s++) {
  const pos = new Float32Array(BURST_PARTICLES * 3);
  const col = new Float32Array(BURST_PARTICLES * 3);
  const vel = new Float32Array(BURST_PARTICLES * 3);
  const bg = new THREE.BufferGeometry();
  bg.setAttribute('position', new THREE.BufferAttribute(pos, 3));
  bg.setAttribute('color', new THREE.BufferAttribute(col, 3));
  bg.setDrawRange(0, 0);
  const bMat = new THREE.PointsMaterial({
    size: 0.4, vertexColors: true, transparent: true, opacity: 1.0,
    blending: THREE.AdditiveBlending, depthWrite: false, sizeAttenuation: true,
    map: pTex
  });
  const points = new THREE.Points(bg, bMat);
  scene.add(points);
  bursts.push({ points, pos, col, vel, active: false, timer: 0 });
}

function triggerBurst(position, color) {
  const slot = bursts.find(b => !b.active);
  if (!slot) return;
  for (let i = 0; i < BURST_PARTICLES; i++) {
    slot.pos[i * 3] = position.x;
    slot.pos[i * 3 + 1] = position.y;
    slot.pos[i * 3 + 2] = position.z;
    const theta = Math.random() * Math.PI * 2;
    const phi = Math.acos(2 * Math.random() - 1);
    const spd = 2 + Math.random() * 3;
    slot.vel[i * 3] = Math.sin(phi) * Math.cos(theta) * spd;
    slot.vel[i * 3 + 1] = Math.sin(phi) * Math.sin(theta) * spd;
    slot.vel[i * 3 + 2] = Math.cos(phi) * spd;
    slot.col[i * 3] = color.r;
    slot.col[i * 3 + 1] = color.g;
    slot.col[i * 3 + 2] = color.b;
  }
  slot.points.geometry.attributes.position.needsUpdate = true;
  slot.points.geometry.attributes.color.needsUpdate = true;
  slot.points.geometry.setDrawRange(0, BURST_PARTICLES);
  slot.active = true;
  slot.timer = 0;
}

function updateBursts(dt) {
  for (const slot of bursts) {
    if (!slot.active) continue;
    slot.timer += dt;
    if (slot.timer > 0.5) {
      slot.points.geometry.setDrawRange(0, 0);
      slot.active = false;
      continue;
    }
    const fade = 1 - dt * 3;
    for (let i = 0; i < BURST_PARTICLES; i++) {
      slot.pos[i * 3] += slot.vel[i * 3] * dt;
      slot.pos[i * 3 + 1] += slot.vel[i * 3 + 1] * dt;
      slot.pos[i * 3 + 2] += slot.vel[i * 3 + 2] * dt;
      slot.col[i * 3] *= fade;
      slot.col[i * 3 + 1] *= fade;
      slot.col[i * 3 + 2] *= fade;
    }
    slot.points.geometry.attributes.position.needsUpdate = true;
    slot.points.geometry.attributes.color.needsUpdate = true;
  }
}

// ---------- NODE PULSE SYSTEM ----------
const roomPulses = {};

function pulseRoom(roomName) {
  roomPulses[roomName] = { t: 0 };
}

function updatePulses(dt) {
  for (const name in roomPulses) {
    const p = roomPulses[name];
    p.t += dt * 3.0;
    if (p.t >= 1.0) {
      const rm = roomMap[name];
      if (rm) rm.mesh.scale.setScalar(1.0);
      delete roomPulses[name];
    } else {
      const rm = roomMap[name];
      if (rm) {
        const ease = 1 - (1 - p.t) * (1 - p.t);
        const s = 1.5 - 0.5 * ease;
        rm.mesh.scale.setScalar(s);
      }
    }
  }
}

// ---------- ANIMATION STATE ----------
let currentTurn = 0;
let turnProgress = 0;
let isPlaying = true;
let speed = 1.0;
let animComplete = turns.length === 0;
let completionMode = false;

function showCompletionHighlight() {
  const usedRooms = new Set();
  for (const key of Object.keys(tunnelTraffic)) {
    const parts = key.split('-');
    usedRooms.add(parts[0]);
    usedRooms.add(parts[1]);
  }
  roomMeshes.forEach(m => {
    if (usedRooms.has(m.userData.roomName)) return;
    const bc = new THREE.Color(m.userData.baseColor);
    m.material.color.copy(bc.multiplyScalar(0.15));
  });
  linkObjects.forEach(line => {
    const key = line.userData.from + '-' + line.userData.to;
    if (tunnelTraffic[key]) {
      line.material.color.set(0x4488cc);
      line.material.opacity = 1.0;
    } else {
      line.material.opacity = 0.05;
    }
  });
  completionMode = true;
}

function clearCompletionHighlight() {
  if (!completionMode) return;
  roomMeshes.forEach(m => {
    m.material.color.set(m.userData.baseColor);
  });
  linkObjects.forEach(line => {
    line.material.color.set(line.userData.baseColor);
    line.material.opacity = line.userData.baseOpacity;
  });
  completionMode = false;
}

function buildTurnAnimations() {
  const anims = [];
  const antPos = {};
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
  const dist = from.distanceTo(to);
  const lift = dist * 0.4;
  const cp1 = new THREE.Vector3().lerpVectors(from, to, 0.25);
  cp1.y += lift;
  const cp2 = new THREE.Vector3().lerpVectors(from, to, 0.75);
  cp2.y += lift;
  const et = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
  return cubicBezier(from, cp1, cp2, to, et);
}

// ---------- ANT PATHS PRE-COMPUTATION ----------
const antPaths = {};
{
  const tempPos = {};
  for (let t = 0; t < turns.length; t++) {
    for (const m of turns[t]) {
      if (!antPaths[m.antId]) antPaths[m.antId] = [SIM_DATA.startName];
      antPaths[m.antId].push(m.room);
      tempPos[m.antId] = m.room;
    }
  }
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
  const intersects = raycaster.intersectObjects(roomMeshes, false);

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

    // Compute neighbors inline
    const nb = [];
    links.forEach(function(l) {
      if (l.from === rName) nb.push(l.to);
      else if (l.to === rName) nb.push(l.from);
    });

    // Compute ants that pass through this room
    const ants = [];
    Object.keys(antPaths).forEach(function(idStr) {
      if (antPaths[idStr].indexOf(rName) !== -1) ants.push(idStr);
    });

    var line1 = rName;
    if (hit.userData.isStart) line1 += '  [START]';
    if (hit.userData.isEnd) line1 += '  [END]';

    var line2 = '';
    if (hit.userData.isStart) {
      line2 = 'All ' + SIM_DATA.antCount + ' ants depart';
    } else if (hit.userData.isEnd) {
      line2 = 'All ' + SIM_DATA.antCount + ' ants arrive';
    } else if (ants.length > 0) {
      var ids = ants.length <= 8 ? ants.join(', ') : ants.slice(0, 7).join(', ') + ' ...';
      line2 = 'Ants: ' + ids + ' (' + ants.length + ')';
    } else {
      line2 = 'No ant traffic';
    }

    var nbStr = nb.length <= 5 ? nb.join(', ') : nb.slice(0, 4).join(', ') + ' +' + (nb.length - 4);
    var line3 = 'Links: ' + nbStr + ' (' + nb.length + ')';

    hoverLabel.textContent = line1 + '\n' + line2 + '\n' + line3;
  } else {
    if (hoveredRoom) {
      resetHighlights();
      hoveredRoom = null;
    }
    hoverLabel.style.display = 'none';
  }
});

function pathsThroughRoom(roomName) {
  const result = new Set();
  Object.entries(antPaths).forEach(([id, path]) => {
    if (path.includes(roomName)) {
      path.forEach(r => result.add(r));
    }
  });
  return result;
}

function tunnelsOnPathsThrough(roomName) {
  const tunnelKeys = new Set();
  Object.values(antPaths).forEach(path => {
    if (!path.includes(roomName)) return;
    for (let i = 0; i < path.length - 1; i++) {
      tunnelKeys.add(path[i] + '-' + path[i + 1]);
      tunnelKeys.add(path[i + 1] + '-' + path[i]);
    }
  });
  return tunnelKeys;
}

// ---------- HIGHLIGHT / RESET ----------
let highlightedTunnelKeys = new Set();
let isPulsing = false;

function highlightRoom(roomName) {
  const pathRooms = pathsThroughRoom(roomName);
  highlightedTunnelKeys = tunnelsOnPathsThrough(roomName);
  isPulsing = true;

  roomMeshes.forEach(m => {
    if (pathRooms.has(m.userData.roomName)) {
      m.material.color.set(0xffffff);
      if (!roomPulses[m.userData.roomName]) {
        m.scale.setScalar(m.userData.roomName === roomName ? 1.4 : 1.2);
      }
    }
  });

  linkObjects.forEach(line => {
    const key = line.userData.from + '-' + line.userData.to;
    if (highlightedTunnelKeys.has(key)) {
      line.material.color.set(0x4488cc);
      line.material.opacity = 1.0;
    }
  });
}

function resetHighlights() {
  isPulsing = false;
  highlightedTunnelKeys.clear();
  roomMeshes.forEach(m => {
    m.material.color.set(m.userData.baseColor);
    if (!roomPulses[m.userData.roomName]) {
      m.scale.setScalar(1.0);
    }
  });
  linkObjects.forEach(line => {
    line.material.color.set(line.userData.baseColor);
    line.material.opacity = line.userData.baseOpacity;
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
  clearCompletionHighlight();
  hideAllAnts();
  clearTrails();
});

btnPrev.addEventListener('click', () => {
  if (currentTurn > 0) {
    clearCompletionHighlight();
    currentTurn--;
    turnProgress = 0;
    animComplete = false;
    rebuildAntsToTurn(currentTurn);
  }
});

btnNext.addEventListener('click', () => {
  if (currentTurn < turns.length - 1) {
    clearCompletionHighlight();
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

const timelineSlider = document.getElementById('timeline-slider');
timelineSlider.max = turns.length;
timelineSlider.addEventListener('input', () => {
  const target = parseInt(timelineSlider.value);
  currentTurn = target;
  turnProgress = 0;
  clearCompletionHighlight();
  if (target >= turns.length) {
    animComplete = true;
    rebuildAntsToTurn(turns.length);
    hideFinishedAnts();
  } else {
    animComplete = false;
    rebuildAntsToTurn(target);
  }
});

function hideAllAnts() {
  Object.values(antMeshes).forEach(m => { m.visible = false; });
}

function rebuildAntsToTurn(targetTurn) {
  hideAllAnts();
  clearTrails();
  const antPos = {};
  for (let t = 0; t < targetTurn; t++) {
    for (const m of turns[t]) {
      antPos[m.antId] = m.room;
    }
  }
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

// ---------- TUNNEL TRAFFIC (brightness by usage) ----------
const tunnelTraffic = {};
{
  const antPos = {};
  for (let t = 0; t < turns.length; t++) {
    for (const m of turns[t]) {
      const from = antPos[m.antId] || SIM_DATA.startName;
      const key1 = from + '-' + m.room;
      const key2 = m.room + '-' + from;
      tunnelTraffic[key1] = (tunnelTraffic[key1] || 0) + 1;
      tunnelTraffic[key2] = (tunnelTraffic[key2] || 0) + 1;
      antPos[m.antId] = m.room;
    }
  }
}

let maxTraffic = 1;
Object.values(tunnelTraffic).forEach(v => { if (v > maxTraffic) maxTraffic = v; });

const idleColor = new THREE.Color(0x1a2a3a);
const activeColor = new THREE.Color(0x4488cc);

linkObjects.forEach(line => {
  const key = line.userData.from + '-' + line.userData.to;
  const traffic = tunnelTraffic[key] || 0;
  const norm = traffic / maxTraffic;
  const c = idleColor.clone().lerp(activeColor, norm);
  const opacity = 0.3 + norm * 0.7;
  line.material.color.copy(c);
  line.material.opacity = opacity;
  line.userData.baseColor = c.getHex();
  line.userData.baseOpacity = opacity;
});

// ---------- FADE IN ----------
setTimeout(function() {
  var fi = document.getElementById('fade-in');
  if (fi) {
    fi.style.opacity = '0';
    fi.addEventListener('transitionend', function() { fi.remove(); });
  }
}, 150);

// ---------- MAIN ANIMATION LOOP ----------
const clock = new THREE.Clock();
let trailTimer = 0;
let elapsed = 0;

function animate() {
  requestAnimationFrame(animate);
  const dt = clock.getDelta();
  elapsed += dt;

  // Camera intro + node fly-in animation
  if (introProgress < 1) {
    introProgress += dt * 0.4;
    if (introProgress > 1) {
      introProgress = 1;
      controls.enabled = true;
    }
    // Circular-out easing (from Particle Globe)
    const ease = Math.sqrt((2 - introProgress) * introProgress);
    camera.position.lerpVectors(camStart, camEnd, ease);
    camera.lookAt(0, 0, 0);

    // Fly-in: lerp room meshes from scatter to final position
    roomMeshes.forEach(m => {
      m.position.lerpVectors(m.userData.scatterPos, m.userData.finalPos, ease);
    });

    // Fade in edges during last 25% of intro
    if (introProgress > 0.75) {
      if (!tunnelGroup.visible) tunnelGroup.visible = true;
      const edgeFade = (introProgress - 0.75) / 0.25;
      linkObjects.forEach(line => {
        line.material.opacity = line.userData.baseOpacity * edgeFade;
      });
    }
  }

  // Ant animation (only after intro completes)
  if (introProgress >= 1 && isPlaying && !animComplete && turns.length > 0 && currentTurn < turns.length) {
    turnProgress += dt * speed * 0.8;

    if (turnProgress >= 1.0) {
      snapAntsToTurnEnd(currentTurn);
      currentTurn++;
      turnProgress = 0;

      if (currentTurn >= turns.length) {
        animComplete = true;
        hideFinishedAnts();
      }
    } else {
      animateAntsInTurn(currentTurn, turnProgress);
    }
  }

  // Trail fade + spawn
  fadeTrails(dt);
  trailTimer += dt;
  if (trailTimer > 0.05) {
    trailTimer = 0;
    const endBound = new Set();
    if (currentTurn < turnAnimations.length) {
      for (const a of turnAnimations[currentTurn]) {
        if (a.toName === SIM_DATA.endName) endBound.add(a.antId);
      }
    }
    Object.entries(antMeshes).forEach(([idStr, mesh]) => {
      const id = parseInt(idStr);
      if (mesh.visible && antCurrentRoom[id] !== SIM_DATA.endName && !endBound.has(id)) {
        addTrailSample(id, mesh.position);
      }
    });
  }

  // Node pulses
  updatePulses(dt);

  // Arrival bursts
  updateBursts(dt);

  // Orbital ring rotation
  orbitalRings.forEach((ring, i) => {
    ring.rotation.z += dt * (0.3 + i * 0.15);
  });

  // Ambient particle drift
  const pArr = ambientParticles.geometry.attributes.position.array;
  const halfSpread = pSpread * 0.5;
  for (let i = 0; i < PARTICLE_COUNT; i++) {
    const v = pVelocities[i];
    pArr[i * 3] += v.vx * dt;
    pArr[i * 3 + 1] += v.vy * dt + Math.sin(elapsed + v.phase) * 0.005;
    pArr[i * 3 + 2] += v.vz * dt;
    for (let j = 0; j < 3; j++) {
      if (pArr[i * 3 + j] > halfSpread) pArr[i * 3 + j] -= pSpread;
      if (pArr[i * 3 + j] < -halfSpread) pArr[i * 3 + j] += pSpread;
    }
  }
  ambientParticles.geometry.attributes.position.needsUpdate = true;

  // Pulse highlighted edges
  if (isPulsing) {
    const pulse = 0.7 + Math.sin(Date.now() * 0.005) * 0.3;
    linkObjects.forEach(line => {
      const key = line.userData.from + '-' + line.userData.to;
      if (highlightedTunnelKeys.has(key)) {
        line.material.opacity = pulse;
      }
    });
  }

  // Completion mode pulse on used edges
  if (completionMode) {
    const cPulse = 0.7 + Math.sin(elapsed * 1.5) * 0.3;
    linkObjects.forEach(line => {
      const key = line.userData.from + '-' + line.userData.to;
      if (tunnelTraffic[key]) line.material.opacity = cPulse;
    });
  }

  // Update turn display + timeline slider
  const displayTurn = animComplete ? turns.length : currentTurn + (turns.length > 0 ? 1 : 0);
  turnValue.textContent = displayTurn + ' / ' + turns.length;
  timelineSlider.value = animComplete ? turns.length : currentTurn;

  controls.update();
  composer.render();
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
    antCurrentRoom[anim.antId] = anim.toName;
    pulseRoom(anim.toName);
    if (anim.toName === SIM_DATA.endName) {
      triggerBurst(anim.toPos, getAntColor(anim.antId));
    }
  }
}

function hideFinishedAnts() {
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
  showCompletionHighlight();
}

// ---------- RESIZE ----------
window.addEventListener('resize', () => {
  camera.aspect = window.innerWidth / window.innerHeight;
  camera.updateProjectionMatrix();
  renderer.setSize(window.innerWidth, window.innerHeight);
  composer.setSize(window.innerWidth, window.innerHeight);
});

// Start
animate();
</script>
</body>
</html>`)

	return sb.String()
}
