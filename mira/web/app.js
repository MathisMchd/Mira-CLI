"use strict";

const state = {
  mode: "list", // "list" | "search"
  query: "",
  limit: 10,
  offset: 0,
  total: 0,
};

const el = {
  createForm: document.getElementById("create-form"),
  createTitle: document.getElementById("create-title"),
  createContent: document.getElementById("create-content"),
  createTags: document.getElementById("create-tags"),
  searchForm: document.getElementById("search-form"),
  searchInput: document.getElementById("search-input"),
  clearSearch: document.getElementById("clear-search"),
  banner: document.getElementById("banner"),
  notesTitle: document.getElementById("notes-title"),
  notesList: document.getElementById("notes-list"),
  pagination: document.getElementById("pagination"),
  refreshBtn: document.getElementById("refresh-btn"),
};

function showError(message) {
  el.banner.textContent = message;
  el.banner.hidden = false;
}

function clearError() {
  el.banner.hidden = true;
  el.banner.textContent = "";
}

async function api(path, options) {
  const res = await fetch(path, options);
  let envelope = null;
  try {
    envelope = await res.json();
  } catch {
    // no body (e.g. 204 No Content)
  }
  if (!res.ok) {
    const message = envelope && envelope.error ? envelope.error.message : `erreur HTTP ${res.status}`;
    throw new Error(message);
  }
  return envelope;
}

function formatDate(iso) {
  const d = new Date(iso);
  return d.toLocaleString("fr-FR", { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit" });
}

function statusLabel(status) {
  if (status === "done") return "enrichie";
  if (status === "failed") return "échec enrichissement";
  return "en cours d'enrichissement";
}

function noteCard(note) {
  const card = document.createElement("div");
  card.className = "note-card";

  const head = document.createElement("div");
  head.className = "note-card-head";

  const title = document.createElement("h3");
  title.className = "note-title";
  title.textContent = note.title;

  const date = document.createElement("span");
  date.className = "note-date";
  date.textContent = formatDate(note.created_at);

  head.append(title, date);
  card.append(head);

  const content = document.createElement("p");
  content.className = "note-content";
  content.textContent = note.content;
  card.append(content);

  if (note.enrichment_status === "done" && note.summary) {
    const summary = document.createElement("p");
    summary.className = "note-summary";
    summary.textContent = `Résumé : ${note.summary}`;
    card.append(summary);
  }

  const footer = document.createElement("div");
  footer.className = "note-footer";

  const status = document.createElement("span");
  status.className = `status status-${note.enrichment_status}`;
  status.textContent = statusLabel(note.enrichment_status);
  footer.append(status);

  if (note.enrichment_status === "done") {
    const score = document.createElement("span");
    score.className = "score";
    score.textContent = `score ${Math.round((note.score || 0) * 100)}%`;
    footer.append(score);
  }

  for (const tag of note.tags || []) {
    const pill = document.createElement("span");
    pill.className = "tag";
    pill.textContent = tag;
    footer.append(pill);
  }

  const del = document.createElement("button");
  del.className = "delete-btn";
  del.textContent = "Supprimer";
  del.addEventListener("click", () => deleteNote(note.id));
  footer.append(del);

  card.append(footer);
  return card;
}

function renderNotes(notes) {
  el.notesList.innerHTML = "";
  if (!notes || notes.length === 0) {
    const empty = document.createElement("p");
    empty.className = "empty";
    empty.textContent = state.mode === "search" ? "Aucun résultat." : "Aucune note pour le moment.";
    el.notesList.append(empty);
    return;
  }
  for (const note of notes) {
    el.notesList.append(noteCard(note));
  }
}

function renderPagination() {
  el.pagination.innerHTML = "";
  if (state.mode !== "list") return;

  const prev = document.createElement("button");
  prev.className = "secondary";
  prev.textContent = "← Précédent";
  prev.disabled = state.offset === 0;
  prev.addEventListener("click", () => {
    state.offset = Math.max(0, state.offset - state.limit);
    loadList();
  });

  const info = document.createElement("span");
  const from = state.total === 0 ? 0 : state.offset + 1;
  const to = Math.min(state.offset + state.limit, state.total);
  info.textContent = `${from}–${to} sur ${state.total}`;

  const next = document.createElement("button");
  next.className = "secondary";
  next.textContent = "Suivant →";
  next.disabled = state.offset + state.limit >= state.total;
  next.addEventListener("click", () => {
    state.offset += state.limit;
    loadList();
  });

  el.pagination.append(prev, info, next);
}

let lastNotes = [];

function hasPending() {
  return lastNotes.some((n) => n.enrichment_status === "pending");
}

async function loadList() {
  try {
    const env = await api(`/api/v1/notes?limit=${state.limit}&offset=${state.offset}`);
    lastNotes = env.data || [];
    state.total = env.meta.total || 0;
    el.notesTitle.textContent = "Notes";
    renderNotes(lastNotes);
    renderPagination();
    clearError();
  } catch (err) {
    showError(err.message);
  }
}

async function loadSearch(query) {
  try {
    const env = await api(`/api/v1/search?q=${encodeURIComponent(query)}`);
    lastNotes = env.data || [];
    el.notesTitle.textContent = `Résultats pour « ${query} »`;
    renderNotes(lastNotes);
    renderPagination();
    clearError();
  } catch (err) {
    showError(err.message);
  }
}

function refresh() {
  if (state.mode === "search") {
    loadSearch(state.query);
  } else {
    loadList();
  }
}

async function deleteNote(id) {
  try {
    await api(`/api/v1/notes/${id}`, { method: "DELETE" });
    refresh();
  } catch (err) {
    showError(err.message);
  }
}

el.createForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  const title = el.createTitle.value.trim();
  const content = el.createContent.value.trim();
  const tags = el.createTags.value
    .split(",")
    .map((t) => t.trim())
    .filter(Boolean);

  try {
    await api("/api/v1/notes", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title, content, tags }),
    });
    el.createForm.reset();
    state.mode = "list";
    state.offset = 0;
    refresh();
  } catch (err) {
    showError(err.message);
  }
});

el.searchForm.addEventListener("submit", (e) => {
  e.preventDefault();
  const q = el.searchInput.value.trim();
  if (!q) return;
  state.mode = "search";
  state.query = q;
  el.clearSearch.hidden = false;
  loadSearch(q);
});

el.clearSearch.addEventListener("click", () => {
  el.searchInput.value = "";
  el.clearSearch.hidden = true;
  state.mode = "list";
  state.offset = 0;
  loadList();
});

el.refreshBtn.addEventListener("click", refresh);

// Tant que des notes affichées sont en cours d'enrichissement, on
// rafraîchit automatiquement pour voir passer leur statut à "done".
setInterval(() => {
  if (hasPending()) refresh();
}, 4000);

loadList();
