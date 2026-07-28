import { useMemo, useRef } from 'react'
import { Canvas, useFrame } from '@react-three/fiber'
import * as THREE from 'three'

const COLS = 38
const ROWS = 22
const GAP = 0.26
const COUNT = COLS * ROWS

const BASE = new THREE.Color('#241f19')
const LIT = new THREE.Color('#6d6055')

const PALETTE = ['#c42d1c', '#c16515', '#c08911', '#276d50', '#146a80'].map(
  (h) => new THREE.Color(h)
)

function Field() {
  const mesh = useRef()
  const dummy = useMemo(() => new THREE.Object3D(), [])
  const scratch = useMemo(() => new THREE.Color(), [])

  const cells = useMemo(() => {
    const out = []
    for (let r = 0; r < ROWS; r++) {
      for (let c = 0; c < COLS; c++) {
        const hit = Math.random() < 0.05
        out.push({
          c,
          r,
          x: (c - (COLS - 1) / 2) * GAP,
          z: (r - (ROWS - 1) / 2) * GAP,
          hit,
          color: hit ? PALETTE[Math.floor(Math.random() * PALETTE.length)] : null,
        })
      }
    }
    return out
  }, [])

  useFrame((state) => {
    const m = mesh.current
    if (!m) return

    const t = state.clock.elapsedTime
    const span = COLS + 16
    const head = ((t * 9) % span) - 8

    for (let i = 0; i < COUNT; i++) {
      const cell = cells[i]

      const d = cell.c - head
      const front = d <= 0 ? Math.max(0, 1 + d / 5) : 0
      const edge = Math.abs(d) < 0.9 ? 1 - Math.abs(d) / 0.9 : 0
      const lift = edge * 0.42

      dummy.position.set(cell.x, lift, cell.z)
      dummy.scale.setScalar(1 + edge * 0.35)
      dummy.updateMatrix()
      m.setMatrixAt(i, dummy.matrix)

      if (cell.hit) {
        scratch.copy(BASE).lerp(cell.color, 0.35 + front * 0.65)
      } else {
        scratch.copy(BASE).lerp(LIT, front * 0.5 + edge * 0.5)
      }
      m.setColorAt(i, scratch)
    }

    m.instanceMatrix.needsUpdate = true
    if (m.instanceColor) m.instanceColor.needsUpdate = true
  })

  return (
    <instancedMesh ref={mesh} args={[null, null, COUNT]}>
      <boxGeometry args={[0.16, 0.16, 0.16]} />
      <meshBasicMaterial toneMapped={false} />
    </instancedMesh>
  )
}

export default function Scene3D() {
  const reduced =
    typeof window !== 'undefined' &&
    window.matchMedia?.('(prefers-reduced-motion: reduce)').matches

  if (reduced) return null

  return (
    <Canvas
      orthographic
      camera={{ position: [7, 6.5, 7], zoom: 74 }}
      dpr={[1, 1.8]}
      gl={{ antialias: true, alpha: true }}
      style={{ pointerEvents: 'none' }}
    >
      <Field />
    </Canvas>
  )
}
