import type { Landmark } from '@/lib/faceGuard'

/** The slice of MediaPipe's FaceLandmarker the app uses (also what tests fake). */
export interface FaceDetector {
  detectForVideo(video: HTMLVideoElement, timestamp: number): {
    faceLandmarks: Landmark[][]
    facialTransformationMatrixes?: { data: ArrayLike<number> }[]
  }
  close(): void
}

// Must match the installed @mediapipe/tasks-vision version. Override to self-host the files.
const WASM_URL = import.meta.env.VITE_MEDIAPIPE_WASM_URL ?? 'https://cdn.jsdelivr.net/npm/@mediapipe/tasks-vision@0.10.3/wasm'
const MODEL_URL =
  import.meta.env.VITE_FACE_MODEL_URL ??
  'https://storage.googleapis.com/mediapipe-models/face_landmarker/face_landmarker/float16/1/face_landmarker.task'

let cached: Promise<FaceDetector> | null = null

/** Loads the face landmarker once (lazy: the library is a separate chunk). GPU first, CPU as fallback. */
export function loadFaceLandmarker(): Promise<FaceDetector> {
  cached ??= (async () => {
    const { FaceLandmarker, FilesetResolver } = await import('@mediapipe/tasks-vision')
    const fileset = await FilesetResolver.forVisionTasks(WASM_URL)
    const make = (delegate: 'GPU' | 'CPU') =>
      FaceLandmarker.createFromOptions(fileset, {
        baseOptions: { modelAssetPath: MODEL_URL, delegate },
        outputFacialTransformationMatrixes: true,
        runningMode: 'VIDEO',
        minFaceDetectionConfidence: 0.3, // lower than the default so a second, partly visible face is still noticed
        minFacePresenceConfidence: 0.3,
        numFaces: 2, // two, so a second person in the frame can be detected and refused
      })
    try {
      return (await make('GPU')) as unknown as FaceDetector
    } catch {
      return (await make('CPU')) as unknown as FaceDetector
    }
  })().catch((e) => {
    cached = null // allow a retry after a network failure
    throw e
  })
  return cached
}
