import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import FaceCapture from './FaceCapture.vue'
import { STABLE_MS } from '@/lib/faceGuard'

const lib = vi.hoisted(() => ({ loadFaceLandmarker: vi.fn() }))
vi.mock('@/lib/mediapipe', () => lib)

// ---- fakes ----
const identity = [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, -40, 1]
const goodFace = [{ x: 0.3, y: 0.25 }, { x: 0.7, y: 0.75 }]
let result: { faceLandmarks: { x: number; y: number }[][]; facialTransformationMatrixes: { data: number[] }[] }
const detector = { detectForVideo: vi.fn(() => result), close: vi.fn() }

const good = () => ({ faceLandmarks: [goodFace], facialTransformationMatrixes: [{ data: identity }] })
const noFace = () => ({ faceLandmarks: [], facialTransformationMatrixes: [] })

let clock = 0
let queue: FrameRequestCallback[] = []
let videoTime = 0
const stop = vi.fn()
const getUserMedia = vi.fn()

/** Runs one animation frame, `dt` ms later, with the video advancing. */
function frame(dt = 100) {
  clock += dt
  videoTime += 0.1
  const cbs = queue
  queue = []
  cbs.forEach((cb) => cb(clock))
}
/** Keeps feeding frames until `ms` have passed. */
function run(ms: number, dt = 100) {
  for (let t = 0; t < ms; t += dt) frame(dt)
}

beforeEach(() => {
  clock = 0
  videoTime = 0
  queue = []
  result = good()
  vi.clearAllMocks()
  vi.spyOn(performance, 'now').mockImplementation(() => clock)
  vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => queue.push(cb))
  vi.stubGlobal('cancelAnimationFrame', () => (queue = []))
  getUserMedia.mockResolvedValue({ getTracks: () => [{ stop }] })
  Object.defineProperty(navigator, 'mediaDevices', { configurable: true, value: { getUserMedia } })
  lib.loadFaceLandmarker.mockResolvedValue(detector)
  Object.defineProperty(HTMLMediaElement.prototype, 'play', { configurable: true, value: vi.fn().mockResolvedValue(undefined) })
  Object.defineProperty(HTMLMediaElement.prototype, 'readyState', { configurable: true, get: () => 4 })
  Object.defineProperty(HTMLMediaElement.prototype, 'currentTime', { configurable: true, get: () => videoTime })
  Object.defineProperty(HTMLVideoElement.prototype, 'videoWidth', { configurable: true, get: () => 2000 })
  Object.defineProperty(HTMLVideoElement.prototype, 'videoHeight', { configurable: true, get: () => 1500 })
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ drawImage: vi.fn() } as unknown as CanvasRenderingContext2D)
  vi.spyOn(HTMLCanvasElement.prototype, 'toBlob').mockImplementation((cb) => cb(new Blob(['jpeg'], { type: 'image/jpeg' })))
})
afterEach(() => vi.unstubAllGlobals())

async function mountActive(props: Record<string, unknown> = {}) {
  const w = mount(FaceCapture, { props: { active: true, ...props } })
  await flushPromises()
  return w
}
/** DOM text after Vue has applied the latest reactive changes. */
const text = async (w: ReturnType<typeof mount>) => {
  await flushPromises()
  return w.get('[data-testid="face-message"]').text()
}

describe('FaceCapture', () => {
  it('opens the front camera and guides the user while no face is visible', async () => {
    result = noFace()
    const w = await mountActive()
    expect(getUserMedia).toHaveBeenCalledWith(expect.objectContaining({ audio: false, video: expect.objectContaining({ facingMode: 'user' }) }))
    run(500)
    expect(await text(w)).toContain('Yuz aniqlanmadi')
    await flushPromises()
    expect(w.find('[data-testid="progress-ring"]').exists()).toBe(false)
    expect(w.emitted('capture')).toBeUndefined()
  })

  it('takes the photo by itself after the face is held steady, then stops the camera', async () => {
    const w = await mountActive()
    run(STABLE_MS / 2)
    expect(await text(w)).toContain("3 soniya to'g'riga qarab turing")
    await flushPromises()
    expect(w.find('[data-testid="progress-ring"]').attributes('stroke-dasharray')).toMatch(/^(4\d|5\d) 100$/) // about half way
    expect(w.emitted('capture')).toBeUndefined()

    run(STABLE_MS / 2 + 400)
    await flushPromises()
    const captured = w.emitted('capture')
    expect(captured).toHaveLength(1)
    const blob = captured![0]![0] as Blob
    expect(blob).toBeInstanceOf(Blob)
    expect(blob.type).toBe('image/jpeg')
    expect(stop).toHaveBeenCalled() // camera released
    run(1000)
    expect(w.emitted('capture')).toHaveLength(1) // exactly once
  })

  it('downsizes the photo to at most 1024 px', async () => {
    const created: HTMLCanvasElement[] = []
    const real = document.createElement.bind(document)
    vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      const el = real(tag)
      if (tag === 'canvas') created.push(el as HTMLCanvasElement)
      return el
    })
    await mountActive().then(() => run(STABLE_MS + 400))
    await flushPromises()
    expect(created.at(-1)!.width).toBe(1024) // 2000x1500 video
    expect(created.at(-1)!.height).toBe(768)
  })

  it('starts the countdown over when the face moves away', async () => {
    const w = await mountActive()
    run(2000)
    result = { faceLandmarks: [[{ x: 0.05, y: 0.25 }, { x: 0.35, y: 0.75 }]], facialTransformationMatrixes: [{ data: identity }] } // off-centre
    run(200)
    expect(await text(w)).toContain('markaziga')
    expect(w.find('[data-testid="progress-ring"]').exists()).toBe(false)
    result = good()
    run(2000) // the reason stays ~1.2 s, then 3 s must be held again: 2 s is not enough
    expect(w.emitted('capture')).toBeUndefined()
    run(2500)
    await flushPromises()
    expect(w.emitted('capture')).toHaveLength(1)
  })

  it('refuses a head that is turned', async () => {
    const yaw = (20 * Math.PI) / 180
    result = { faceLandmarks: [goodFace], facialTransformationMatrixes: [{ data: [Math.cos(yaw), 0, -Math.sin(yaw), 0, 0, 1, 0, 0, Math.sin(yaw), 0, Math.cos(yaw), 0, 0, 0, -40, 1] }] }
    const w = await mountActive()
    run(STABLE_MS + 1000)
    expect(await text(w)).toContain("Boshingizni to'g'ri tuting")
    expect(w.emitted('capture')).toBeUndefined()
  })

  it('refuses when a second person is in the frame', async () => {
    result = { faceLandmarks: [goodFace, goodFace], facialTransformationMatrixes: [{ data: identity }] }
    const w = await mountActive()
    run(STABLE_MS + 500)
    expect(await text(w)).toContain('faqat bitta odam')
    expect(w.emitted('capture')).toBeUndefined()
  })

  it('holds the countdown while the parent is not ready (GPS) and resumes afterwards', async () => {
    const w = await mountActive({ blockedMessage: 'Joylashuv aniqlanishini kuting...' })
    run(STABLE_MS + 1000)
    expect(await text(w)).toContain('Joylashuv aniqlanishini kuting')
    expect(w.emitted('capture')).toBeUndefined()

    await w.setProps({ blockedMessage: '' })
    run(STABLE_MS + 500)
    await flushPromises()
    expect(w.emitted('capture')).toHaveLength(1)
  })

  it('shows a clear message and a retry button when the camera is denied', async () => {
    getUserMedia.mockRejectedValueOnce(Object.assign(new Error('x'), { name: 'NotAllowedError' }))
    const w = await mountActive()
    expect(await text(w)).toContain('Kameraga ruxsat berilmagan')
    await w.get('button').trigger('click')
    await flushPromises()
    expect(getUserMedia).toHaveBeenCalledTimes(2) // retry asks again
    expect(w.find('button').exists()).toBe(false)
  })

  it('reports a failed model download and can retry', async () => {
    lib.loadFaceLandmarker.mockRejectedValueOnce(new Error('offline'))
    const w = await mountActive()
    expect(await text(w)).toContain('modeli yuklanmadi')
    expect(getUserMedia).not.toHaveBeenCalled()
    await w.get('button').trigger('click')
    await flushPromises()
    expect(getUserMedia).toHaveBeenCalledOnce()
  })

  it('releases the camera when it becomes inactive or is unmounted', async () => {
    const w = await mountActive()
    await w.setProps({ active: false })
    expect(stop).toHaveBeenCalled()
    stop.mockClear()
    await w.setProps({ active: true })
    await flushPromises()
    w.unmount()
    expect(stop).toHaveBeenCalled()
  })

  it('does not start the camera while inactive', async () => {
    mount(FaceCapture, { props: { active: false } })
    await flushPromises()
    expect(getUserMedia).not.toHaveBeenCalled()
  })
})
