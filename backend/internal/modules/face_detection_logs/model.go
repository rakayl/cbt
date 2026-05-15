package face_detection_logs

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type FaceDetectionLog struct{ shared.Record }

const TableName = "face_detection_logs"
const PermissionRead = "face.detection.logs:read"
const PermissionWrite = "face.detection.logs:write"
