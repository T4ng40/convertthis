const SUPPORTED = ["wav", "mp3"];

const el = (id) => document.getElementById(id);
const dropzone = el("dropzone");
const fileInput = el("file-input");
const fileRow = el("file-row");
const fileName = el("file-name");
const fileMeta = el("file-meta");
const fileClear = el("file-clear");
const targetRow = el("target-row");
const targetOptions = el("target-options");
const convertBtn = el("convert-btn");
const convertLabel = el("convert-label");
const convertSpinner = el("convert-spinner");
const result = el("result");
const resultName = el("result-name");
const downloadLink = el("download-link");
const errorBox = el("error");

let selectedFile = null;
let sourceFormat = null;
let targetFormat = null;

const root = document.documentElement;
if (localStorage.getItem("theme") === "light") root.classList.remove("dark");
el("theme-toggle").addEventListener("click", () => {
  root.classList.toggle("dark");
  localStorage.setItem(
    "theme",
    root.classList.contains("dark") ? "dark" : "light",
  );
});

function ext(name) {
  const i = name.lastIndexOf(".");
  return i === -1 ? "" : name.slice(i + 1).toLowerCase();
}

function humanSize(bytes) {
  if (bytes < 1024) return bytes + " B";
  const units = ["KB", "MB", "GB"];
  let n = bytes / 1024,
    u = 0;
  while (n >= 1024 && u < units.length - 1) {
    n /= 1024;
    u++;
  }
  return n.toFixed(1) + " " + units[u];
}

function hide(node) {
  node.classList.add("hidden");
}
function show(node, display = "block") {
  node.classList.remove("hidden");
  node.style.display = display;
}

function showError(msg) {
  errorBox.textContent = msg;
  errorBox.classList.remove("hidden");
}

function resetResultAndError() {
  hide(result);
  hide(errorBox);
}

function selectFile(file) {
  resetResultAndError();
  const e = ext(file.name);
  if (!SUPPORTED.includes(e)) {
    showError(
      `Unsupported file type ".${e}". Please choose a WAV or MP3 file.`,
    );
    return;
  }

  selectedFile = file;
  sourceFormat = e;
  targetFormat = null;

  fileName.textContent = file.name;
  fileMeta.textContent = `${e.toUpperCase()} · ${humanSize(file.size)}`;
  hide(dropzone);
  fileRow.classList.remove("hidden");
  fileRow.style.display = "flex";

  renderTargets();
  updateConvertButton();
}

function renderTargets() {
  const targets = SUPPORTED.filter((f) => f !== sourceFormat);
  targetOptions.innerHTML = "";
  for (const fmt of targets) {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.dataset.fmt = fmt;
    btn.className =
      "target-opt rounded-xl border border-slate-200 px-4 py-3 text-sm font-semibold uppercase transition hover:border-brand-400 dark:border-slate-700";
    btn.textContent = fmt;
    btn.addEventListener("click", () => {
      targetFormat = fmt;
      for (const b of targetOptions.children) {
        const active = b === btn;
        b.classList.toggle("border-brand-500", active);
        b.classList.toggle("bg-brand-500/10", active);
        b.classList.toggle("text-brand-500", active);
      }
      updateConvertButton();
    });
    targetOptions.appendChild(btn);
  }
  if (targets.length === 1) targetOptions.firstChild.click();

  targetRow.classList.remove("hidden");
}

function clearFile() {
  selectedFile = null;
  sourceFormat = null;
  targetFormat = null;
  fileInput.value = "";
  hide(fileRow);
  hide(targetRow);
  resetResultAndError();
  show(dropzone, "flex");
  updateConvertButton();
}

function updateConvertButton() {
  convertBtn.disabled = !(selectedFile && targetFormat);
  convertLabel.textContent = targetFormat
    ? `Convert to ${targetFormat.toUpperCase()}`
    : "Convert";
}

fileInput.addEventListener("change", (e) => {
  if (e.target.files.length) selectFile(e.target.files[0]);
});
fileClear.addEventListener("click", (e) => {
  e.preventDefault();
  clearFile();
});

["dragenter", "dragover"].forEach((evt) =>
  dropzone.addEventListener(evt, (e) => {
    e.preventDefault();
    dropzone.classList.add("border-brand-400", "bg-brand-500/5");
  }),
);
["dragleave", "drop"].forEach((evt) =>
  dropzone.addEventListener(evt, (e) => {
    e.preventDefault();
    dropzone.classList.remove("border-brand-400", "bg-brand-500/5");
  }),
);
dropzone.addEventListener("drop", (e) => {
  if (e.dataTransfer.files.length) selectFile(e.dataTransfer.files[0]);
});

convertBtn.addEventListener("click", async () => {
  if (!selectedFile || !targetFormat) return;
  resetResultAndError();
  setLoading(true);

  const form = new FormData();
  form.append("file", selectedFile);
  form.append("target", targetFormat);

  try {
    const res = await fetch("/api/convert", { method: "POST", body: form });
    const data = await res.json();
    if (!res.ok)
      throw new Error(data.error || `Request failed (${res.status})`);

    resultName.textContent = data.filename;
    downloadLink.href = data.download;
    downloadLink.setAttribute("download", data.filename);
    result.classList.remove("hidden");
  } catch (err) {
    showError(err.message || "Something went wrong.");
  } finally {
    setLoading(false);
  }
});

function setLoading(on) {
  convertBtn.disabled = on || !(selectedFile && targetFormat);
  convertSpinner.classList.toggle("hidden", !on);
  convertLabel.textContent = on
    ? "Converting…"
    : targetFormat
      ? `Convert to ${targetFormat.toUpperCase()}`
      : "Convert";
}
