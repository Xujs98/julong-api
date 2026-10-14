export const meta = {
  apiVersion: 1,
  key: "suanliai-video",
  name: "Suanli AI Video",
  icon: "text:算力",
  version: "1.0.1",
  author: { name: "Julong" },
  description: { en: "Video generation through the Suanli AI unified gateway", zh: "通过算力 AI 统一网关生成视频" },
  baseUrl: "https://suanliai.top",
  website: "https://suanliai.top/docs",
  upstreams: ["vendor", "new_api"],
  // Additional administrator-configured channel model names are bound by the host.
  models: ["专享sd2.0(fast933/4-15秒/720p)", "专享sd2.5(30图10音/4-30秒/720p)"],
  fetchMode: "per_task",
  protocols: ["openai_video", { name: "openai_responses", supports: ["stream", "sync", "background"] }],
  usageSchema: {
    seconds: { type: "number", unit: "second", description: { en: "Video generation unit price", zh: "视频生成单价" } },
    tasks: { type: "number", unit: "count", description: { en: "Video task unit price", zh: "视频任务单价" } },
  },
  usageExamples: [
    { label: "ten seconds", facts: { seconds: 10, tasks: 1 } },
  ],
};

const aliases = {
  seconds: ["duration", "duration_seconds"],
  aspect_ratio: ["ratio", "aspectRatio"],
  resolution: ["resolution_name"],
  images: ["image", "image_url", "image_urls", "reference_images", "referenceImages", "image_refs", "input_reference"],
  videos: ["video", "video_url", "video_urls", "reference_videos", "referenceVideos", "video_refs"],
  audios: ["audio", "audio_url", "audio_urls", "reference_audios", "referenceAudios", "audio_refs"],
};
function asList(value) {
  if (value === undefined || value === null) return [];
  return Array.isArray(value) ? value : [value];
}
function mergeLists(values) {
  const merged = [], seen = new Set();
  for (const value of values.flatMap(asList)) {
    const key = typeof value === "string" ? value : JSON.stringify(value);
    if (!seen.has(key)) { seen.add(key); merged.push(value); }
  }
  return merged;
}
function normalize(input, model) {
  if (!input || typeof input !== "object" || Array.isArray(input)) throw new Error("JSON object required");
  const req = Object.assign({}, input, { model });
  for (const key of Object.keys(aliases)) {
    const fields = [key, ...aliases[key]];
    const present = fields.filter((field) => req[field] !== undefined);
    if (key === "images" || key === "videos" || key === "audios") {
      if (present.length > 0) req[key] = mergeLists(present.map((field) => req[field]));
    } else if (req[key] === undefined) {
      for (const alias of aliases[key]) if (req[alias] !== undefined) { req[key] = req[alias]; break; }
    }
    for (const alias of aliases[key]) delete req[alias];
  }
  if (typeof req.prompt !== "string" || !req.prompt.trim()) throw new Error("field prompt is required");
  if (req.seconds !== undefined) {
    const seconds = Number(req.seconds);
    if (!Number.isFinite(seconds) || seconds < 0 || seconds > 3600 || seconds !== Math.trunc(seconds)) throw new Error("seconds must be an integer between 0 and 3600");
    if (seconds === 0) delete req.seconds;
    else req.seconds = seconds;
  }
  for (const key of ["images", "videos", "audios"]) {
    if (req[key] === undefined) continue;
    let values = req[key];
    if (typeof values === "string") values = [values];
    if (!Array.isArray(values) || values.length > 128) throw new Error(key + " must be an array with at most 128 references");
    for (const value of values) if (typeof value !== "string" && !(value && typeof value === "object" && value.__fileRef)) throw new Error(key + " must contain media links or data URLs");
    req[key] = values;
  }
  for (const key of ["face", "generate_audio"]) {
    if (req[key] === "true") req[key] = true;
    if (req[key] === "false") req[key] = false;
    if (req[key] !== undefined && typeof req[key] !== "boolean") throw new Error(key + " must be a boolean");
  }
  delete req.stream;
  delete req.background;
  return req;
}
function decode(ctx) {
  const body = ctx.body || {};
  let req;
  if (body.kind === "json") req = normalize(body.value, ctx.model);
  else if (body.kind === "multipart") {
    const values = {}, fields = body.fields || {};
    for (const field of Object.keys(fields)) {
      const key = field.endsWith("[]") ? field.slice(0, -2) : field;
      if (["images", "videos", "audios"].includes(key)) {
        let items = fields[field];
        if (items.length === 1 && typeof items[0] === "string" && items[0].trim().startsWith("[")) {
          try { items = JSON.parse(items[0]); } catch (e) { throw new Error(key + " must be valid JSON or repeated fields"); }
        }
        if (!Array.isArray(items)) throw new Error(key + " must be an array");
        values[key] = (values[key] || []).concat(items);
      } else {
        if (fields[field].length !== 1) throw new Error(field + " must be provided once");
        values[key] = fields[field][0];
      }
    }
    for (const file of body.files || []) {
      const key = file.field.replace(/\[\]$/, "");
      if (!["images", "videos", "audios", "input_reference"].includes(key)) throw new Error("unexpected file field: " + file.field);
      if (file.size > 50 * 1024 * 1024) throw new Error("reference file exceeds 50 MB");
      const type = key === "input_reference" ? "images" : key;
      values[type] = (values[type] || []).concat([{ __fileRef: file.ref, encoding: "dataUrl", mimeType: file.mimeType, maxBytes: 50 * 1024 * 1024 }]);
    }
    req = normalize(values, ctx.model);
  } else throw new Error("JSON or multipart body required");
  const action = req.videos && req.videos.length ? "reference_to_video" : req.images && req.images.length ? "image_to_video" : "text_to_video";
  return { kind: "submit", model: ctx.model, action, requestBody: req };
}
function base(ctx) { return ctx.baseUrl.replace(/\/+$/, "").replace(/\/v1$/, ""); }
function stateFor(previous, body, seconds) {
  const state = Object.assign({}, previous || {});
  if (seconds !== undefined) state.seconds = seconds;
  if (body && typeof body.origin_task_id === "string") state.origin_task_id = body.origin_task_id;
  return state;
}
function result(ctx, body) {
  if (!body || typeof body !== "object" || Array.isArray(body)) return { status: "UNKNOWN" };
  const status = { queued: "QUEUED", pending: "QUEUED", in_progress: "IN_PROGRESS", archiving: "IN_PROGRESS", completed: "SUCCESS", failed: "FAILURE" }[body.status] || "UNKNOWN";
  const progress = body.status === "archiving" ? 99 : Number(body.progress);
  const output = { status, progress: Number.isFinite(progress) ? String(Math.max(0, Math.min(100, progress))) + "%" : status === "SUCCESS" ? "100%" : "0%", state: stateFor(ctx.state, body) };
  if (status === "FAILURE") output.reason = body.error && body.error.message || "Video generation failed";
  if (status === "SUCCESS") {
    output.url = "/v1/videos/" + ctx.publicTaskId + "/content";
    const remote = body.result_url || body.url || body.video_url || body.download_url;
    if (typeof remote === "string") output.remoteUrl = remote;
  }
  return output;
}
export function buildSubmitRequest(ctx) {
  return { url: base(ctx) + "/v1/videos", method: "POST", headers: { Authorization: ctx.authHeader || "Bearer " + ctx.apiKey, "Content-Type": "application/json" }, body: Object.assign({}, ctx.requestBody, { model: ctx.upstreamModel || ctx.model }) };
}
export function parseSubmitResponse(ctx, response) {
  const body = response.body;
  if (!body || typeof body !== "object" || typeof body.id !== "string" || !body.id) throw new Error("video response id is required");
  const state = stateFor(null, body, ctx.requestBody.seconds);
  const parsed = { taskId: body.id, taskData: body, state };
  if (["completed", "failed"].includes(body.status)) parsed.immediate = result(Object.assign({}, ctx, { state }), body);
  return parsed;
}
export function buildQueryRequest(ctx) { return { url: base(ctx) + "/v1/videos/" + encodeURIComponent(ctx.taskId), method: "GET", headers: { Authorization: ctx.authHeader || "Bearer " + ctx.apiKey } }; }
export function parseTaskResult(ctx, body) { return result(ctx, body); }
export function extractUsage(ctx) { return { seconds: ctx.requestBody.seconds, tasks: 1 }; }
export function extractUsageOnComplete(ctx, task, body) {
  const requested = ctx.state && ctx.state.seconds;
  const measured = body && (body.seconds === undefined ? body.duration : body.seconds);
  const seconds = measured === undefined ? requested : Number(measured);
  if (!Number.isFinite(seconds) || seconds <= 0 || seconds > 3600) return null;
  return { seconds, tasks: 1 };
}
export function listArtifacts(task) { return task.status === "SUCCESS" ? [{ key: "video", type: "video", mimeType: "video/mp4" }] : []; }
export function buildContentRequest(ctx) {
  const data = ctx.data || {};
  const resultUrl = data.result_url || data.url || data.video_url || data.download_url;
  if (typeof resultUrl === "string" && resultUrl.trim()) {
    let url = resultUrl.trim();
    if (url.startsWith("/") && !url.startsWith("//")) url = base(ctx).replace(/^(https?:\/\/[^/]+).*$/, "$1") + url;
    if (!/^https?:\/\//i.test(url)) throw new Error("Invalid video result URL");
    return { url, method: ctx.clientRequest.method, credentialless: true };
  }
  return { url: base(ctx) + "/v1/videos/" + encodeURIComponent(ctx.upstreamTaskId) + "/content", method: ctx.clientRequest.method, headers: { Authorization: ctx.authHeader || "Bearer " + ctx.apiKey } };
}
export const protocols = {
  openai_video: {
    decodeRequest: decode,
    render(ctx, task) { return task.data && typeof task.data === "object" ? Object.assign({}, task.data, task.status === "SUCCESS" ? { url: "/v1/videos/" + task.task_id + "/content", video_url: "/v1/videos/" + task.task_id + "/content", result_url: "/v1/videos/" + task.task_id + "/content", download_url: "/v1/videos/" + task.task_id + "/content" } : {}) : {}; },
  },
  openai_responses: {
    decodeRequest(ctx) {
      const source = ctx.body && ctx.body.value;
      if (!source || (typeof source.input !== "string" && !Array.isArray(source.input))) throw new Error("input must be a string or array");
      const texts = [], images = [];
      if (typeof source.input === "string") texts.push(source.input);
      else for (const item of source.input) {
        if (typeof item === "string") { texts.push(item); continue; }
        for (const part of item.content || [item]) {
          if (part.type === "input_text" || part.type === "text") texts.push(part.text || "");
          if (part.type === "input_image") images.push(typeof part.image_url === "string" ? part.image_url : part.image_url.url);
        }
      }
      return decode(Object.assign({}, ctx, { body: { kind: "json", value: Object.assign({}, source, { prompt: texts.join("\n"), images: source.images || images }) } }));
    },
    renderFinal(ctx) { const media = ctx.artifacts && ctx.artifacts.video; if (!media) throw new Error("video artifact is unavailable"); return [{ role: "assistant", content: [{ type: "output_text", text: '<video controls src="' + media.url.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;") + '"></video>' }] }]; },
    renderEvents(ctx, task, previous) { if (task.status !== "SUCCESS" || previous) return { state: previous || null, events: [] }; return { state: true, events: [{ type: "output_text", text: protocols.openai_responses.renderFinal(ctx)[0].content[0].text }] }; },
  },
};
