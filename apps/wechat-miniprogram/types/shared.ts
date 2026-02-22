export type IntentLabel =
  | "FEEDING"
  | "SEEK_ATTENTION"
  | "WANT_PLAY"
  | "WANT_DOOR_OPEN"
  | "DEFENSIVE_ALERT"
  | "RELAX_SLEEP"
  | "CURIOUS_OBSERVE"
  | "UNCERTAIN";

export type Level3 = "LOW" | "MID" | "HIGH";

export interface State3D {
  tension: Level3;
  arousal: Level3;
  comfort: Level3;
}

export interface InferenceResult {
  intentTop3: Array<{ label: IntentLabel; prob: number }>;
  state: State3D;
  confidence: number;
  source: "EDGE" | "CLOUD";
  evidence: string[];
  copyStyleVersion: string;
  edgeMeta?: EdgeMeta;
  risk?: PainRisk;
}

export interface CopyBlock {
  catLine: string;
  evidence: string;
  shareTitle: string;
}

export interface FinalizeResponse {
  sampleId: string;
  result: InferenceResult;
  copy: CopyBlock;
  needFeedback: boolean;
  fallbackUsed: boolean;
}

export interface EdgeRuntime {
  engine: string;
  modelVersion: string;
  modelHash?: string;
  inputShape?: string;
  loadMs: number;
  inferMs: number;
  deviceModel: string;
  failureCode?: string;
  failureReason?: string;
}

export interface EdgeMeta extends EdgeRuntime {
  fallbackUsed: boolean;
  usedEdgeResult: boolean;
}

export interface PainRisk {
  painRiskScore: number;
  painRiskLevel: "LOW" | "MID" | "HIGH";
  riskEvidence: string[];
  disclaimer: string;
}
