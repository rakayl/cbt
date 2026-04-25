"""
CBT Enterprise — AI Proctoring Service
FastAPI + face_recognition + YOLO (ultralytics)

Endpoints:
  POST /detect/face      — detect faces in a frame
  POST /verify/face      — compare frame vs baseline embedding
  POST /embedding/generate — generate baseline face embedding
  POST /detect/objects   — YOLO object detection (phone, etc.)
"""

import base64
import io
import logging
from typing import Optional

import numpy as np
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from PIL import Image
from pydantic import BaseModel

# Optional imports — gracefully degrade if not installed
try:
    import face_recognition
    FACE_RECOGNITION_AVAILABLE = True
except ImportError:
    FACE_RECOGNITION_AVAILABLE = False
    logging.warning("face_recognition not installed — face endpoints will return mock data")

try:
    from ultralytics import YOLO
    yolo_model = YOLO("yolov8n.pt")
    YOLO_AVAILABLE = True
except Exception:
    YOLO_AVAILABLE = False
    yolo_model = None
    logging.warning("YOLO not available — object detection will return empty results")

# ── App setup ──────────────────────────────────────────────────────────────────

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("proctoring")

app = FastAPI(title="CBT AI Proctoring Service", version="1.0.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

FACE_MISMATCH_THRESHOLD = 0.6   # cosine distance; lower = more similar
YOLO_CONFIDENCE         = 0.4
PHONE_CLASSES           = {"cell phone", "laptop", "book", "tablet"}   # COCO class names

# ── Schemas ───────────────────────────────────────────────────────────────────

class DetectRequest(BaseModel):
    image_base64: str
    attempt_id: Optional[str] = None

class VerifyRequest(BaseModel):
    image_base64: str
    base_embedding: list[float]
    attempt_id: Optional[str] = None

class EmbeddingRequest(BaseModel):
    image_base64: str

# ── Helpers ───────────────────────────────────────────────────────────────────

def decode_image(b64: str) -> np.ndarray:
    """Decode base64 JPEG/PNG to numpy RGB array."""
    data = base64.b64decode(b64)
    img  = Image.open(io.BytesIO(data)).convert("RGB")
    return np.array(img)


def get_face_locations(img_array: np.ndarray):
    """Return list of (top, right, bottom, left) face bounding boxes."""
    if not FACE_RECOGNITION_AVAILABLE:
        return []
    return face_recognition.face_locations(img_array, model="hog")


def get_face_embeddings(img_array: np.ndarray, locations=None):
    """Return list of 128-d face embedding vectors."""
    if not FACE_RECOGNITION_AVAILABLE:
        return []
    return face_recognition.face_encodings(img_array, known_face_locations=locations)


# ── Endpoints ─────────────────────────────────────────────────────────────────

@app.get("/health")
def health():
    return {
        "status": "ok",
        "face_recognition": FACE_RECOGNITION_AVAILABLE,
        "yolo": YOLO_AVAILABLE,
    }


@app.post("/detect/face")
def detect_face(req: DetectRequest):
    try:
        img = decode_image(req.image_base64)
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"Invalid image: {e}")

    locations = get_face_locations(img)
    face_count = len(locations)
    has_face   = face_count > 0

    bbox = None
    if has_face:
        top, right, bottom, left = locations[0]
        bbox = [left, top, right - left, bottom - top]

    # Simple confidence: fraction of image area covered by largest face
    confidence = 0.0
    if has_face:
        h, w = img.shape[:2]
        top, right, bottom, left = locations[0]
        face_area  = (right - left) * (bottom - top)
        image_area = w * h
        confidence = min(face_area / max(image_area, 1) * 10, 1.0)

    logger.info(f"[detect_face] attempt={req.attempt_id} faces={face_count}")

    return {
        "face_count":   face_count,
        "has_face":     has_face,
        "confidence":   round(confidence, 3),
        "bounding_box": bbox,
    }


@app.post("/verify/face")
def verify_face(req: VerifyRequest):
    try:
        img = decode_image(req.image_base64)
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"Invalid image: {e}")

    if not FACE_RECOGNITION_AVAILABLE:
        return {"match": True, "similarity": 0.99, "threshold": FACE_MISMATCH_THRESHOLD}

    locations   = get_face_locations(img)
    if not locations:
        return {"match": False, "similarity": 0.0, "threshold": FACE_MISMATCH_THRESHOLD}

    embeddings  = get_face_embeddings(img, locations)
    if not embeddings:
        return {"match": False, "similarity": 0.0, "threshold": FACE_MISMATCH_THRESHOLD}

    base = np.array(req.base_embedding, dtype=np.float64)
    curr = np.array(embeddings[0], dtype=np.float64)

    # face_recognition distance (euclidean) — 0 = identical, >0.6 = different
    distance   = float(np.linalg.norm(base - curr))
    similarity = max(0.0, 1.0 - distance)
    match      = distance <= FACE_MISMATCH_THRESHOLD

    logger.info(f"[verify_face] attempt={req.attempt_id} dist={distance:.3f} match={match}")

    return {
        "match":     match,
        "similarity": round(similarity, 4),
        "threshold": FACE_MISMATCH_THRESHOLD,
    }


@app.post("/embedding/generate")
def generate_embedding(req: EmbeddingRequest):
    try:
        img = decode_image(req.image_base64)
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"Invalid image: {e}")

    if not FACE_RECOGNITION_AVAILABLE:
        # Return random 128-d mock embedding for dev mode
        mock = (np.random.rand(128) * 0.2).tolist()
        return {"embedding": mock, "success": True}

    locations  = get_face_locations(img)
    if not locations:
        return {"embedding": [], "success": False}

    embeddings = get_face_embeddings(img, locations)
    if not embeddings:
        return {"embedding": [], "success": False}

    return {"embedding": embeddings[0].tolist(), "success": True}


@app.post("/detect/objects")
def detect_objects(req: DetectRequest):
    if not YOLO_AVAILABLE or yolo_model is None:
        return {"objects": []}

    try:
        img = decode_image(req.image_base64)
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"Invalid image: {e}")

    results  = yolo_model.predict(img, conf=YOLO_CONFIDENCE, verbose=False)
    detected = []
    for r in results:
        for box in r.boxes:
            cls_name = r.names[int(box.cls)]
            if cls_name in PHONE_CLASSES:
                detected.append(cls_name)

    logger.info(f"[detect_objects] attempt={req.attempt_id} objects={detected}")
    return {"objects": list(set(detected))}


# ── Run ───────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=5001, log_level="info")
