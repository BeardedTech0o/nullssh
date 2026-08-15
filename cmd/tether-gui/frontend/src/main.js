import './style.css';
import {Terminal} from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import {FitAddon} from '@xterm/addon-fit';

import {
    ListConnections,
    AddConnection,
    UpdateConnection,
    DeleteConnection,
    BrowseIdentityFile,
    OpenSession,
    WriteToSession,
    ResizeSession,
    CloseSession,
} from '../wailsjs/go/main/App';
import {EventsOn, EventsOff} from '../wailsjs/runtime/runtime';

const connectionListEl = document.getElementById('connection-list');
const tabBarEl = document.getElementById('tab-bar');
const terminalAreaEl = document.getElementById('terminal-area');
const emptyStateEl = document.getElementById('empty-state');
const addBtn = document.getElementById('add-btn');
const addModal = document.getElementById('add-modal');
const addModalTitle = document.getElementById('add-modal-title');
const addForm = document.getElementById('add-form');
const addCancel = document.getElementById('add-cancel');
const addError = document.getElementById('add-error');
const browseIdentityBtn = document.getElementById('browse-identity-btn');

// Name of the connection currently being edited, or null when the modal is
// in "add" mode.
let editingName = null;

/** @type {Map<string, {name: string, term: Terminal, fit: FitAddon, pane: HTMLElement, tab: HTMLElement}>} */
const sessions = new Map();
let activeSessionId = null;

async function refreshConnections() {
    const conns = await ListConnections();
    connectionListEl.innerHTML = '';

    if (!conns || conns.length === 0) {
        const li = document.createElement('li');
        li.className = 'empty-list';
        li.textContent = 'No saved connections yet.';
        connectionListEl.appendChild(li);
        return;
    }

    for (const c of conns) {
        const li = document.createElement('li');
        li.className = 'connection-item';

        const info = document.createElement('div');
        const name = document.createElement('span');
        name.className = 'connection-name';
        name.textContent = c.name;
        const target = document.createElement('span');
        target.className = 'connection-target';
        target.textContent = `${c.user}@${c.host}:${c.port}`;
        info.appendChild(name);
        info.appendChild(target);

        const edit = document.createElement('button');
        edit.className = 'connection-edit';
        edit.textContent = '✎';
        edit.title = 'Edit connection';
        edit.addEventListener('click', (e) => {
            e.stopPropagation();
            openEditModal(c);
        });

        const del = document.createElement('button');
        del.className = 'connection-delete';
        del.textContent = '✕';
        del.title = 'Delete connection';
        del.addEventListener('click', async (e) => {
            e.stopPropagation();
            await DeleteConnection(c.name);
            await refreshConnections();
        });

        const actions = document.createElement('div');
        actions.className = 'connection-actions';
        actions.appendChild(edit);
        actions.appendChild(del);

        li.appendChild(info);
        li.appendChild(actions);
        li.addEventListener('click', () => openConnection(c.name));

        connectionListEl.appendChild(li);
    }
}

function updateEmptyState() {
    const hasSessions = sessions.size > 0;
    emptyStateEl.classList.toggle('hidden', hasSessions);
    terminalAreaEl.classList.toggle('hidden', !hasSessions);
}

async function openConnection(name) {
    let id;
    try {
        id = await OpenSession(name);
    } catch (err) {
        alert(`Failed to open session: ${err}`);
        return;
    }

    const pane = document.createElement('div');
    pane.className = 'terminal-pane';
    terminalAreaEl.appendChild(pane);

    const term = new Terminal({
        convertEol: true,
        theme: {background: '#12181f'},
        fontSize: 13,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(pane);

    term.onData((data) => {
        WriteToSession(id, data).catch((err) => console.error(err));
    });
    term.onResize(({cols, rows}) => {
        ResizeSession(id, cols, rows).catch((err) => console.error(err));
    });

    const tab = document.createElement('div');
    tab.className = 'tab';
    const label = document.createElement('span');
    label.textContent = name;
    const closeBtn = document.createElement('button');
    closeBtn.className = 'tab-close';
    closeBtn.textContent = '✕';
    closeBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        CloseSession(id).catch((err) => console.error(err));
    });
    tab.appendChild(label);
    tab.appendChild(closeBtn);
    tab.addEventListener('click', () => activateSession(id));
    tabBarEl.appendChild(tab);

    EventsOn(`session:${id}:data`, (chunk) => {
        term.write(chunk);
    });
    EventsOn(`session:${id}:closed`, () => {
        removeSession(id);
    });

    sessions.set(id, {name, term, fit, pane, tab});
    activateSession(id);
    updateEmptyState();

    // Give the pane a layout pass before fitting.
    requestAnimationFrame(() => fit.fit());
}

function activateSession(id) {
    if (!sessions.has(id)) return;
    activeSessionId = id;
    for (const [sid, s] of sessions) {
        const isActive = sid === id;
        s.pane.classList.toggle('active', isActive);
        s.tab.classList.toggle('active', isActive);
    }
    const s = sessions.get(id);
    s.fit.fit();
    s.term.focus();
}

function removeSession(id) {
    const s = sessions.get(id);
    if (!s) return;

    EventsOff(`session:${id}:data`);
    EventsOff(`session:${id}:closed`);

    s.term.dispose();
    s.pane.remove();
    s.tab.remove();
    sessions.delete(id);

    if (activeSessionId === id) {
        activeSessionId = null;
        const next = sessions.keys().next().value;
        if (next) activateSession(next);
    }
    updateEmptyState();
}

window.addEventListener('resize', () => {
    if (activeSessionId && sessions.has(activeSessionId)) {
        sessions.get(activeSessionId).fit.fit();
    }
});

// Add/edit connection modal

browseIdentityBtn.addEventListener('click', async () => {
    try {
        const path = await BrowseIdentityFile();
        if (path) {
            addForm.identityFile.value = path;
        }
    } catch (err) {
        addError.textContent = String(err);
        addError.classList.remove('hidden');
    }
});

addBtn.addEventListener('click', () => {
    editingName = null;
    addModalTitle.textContent = 'Add connection';
    addError.classList.add('hidden');
    addForm.reset();
    addForm.port.value = 22;
    addModal.classList.remove('hidden');
    addForm.name.focus();
});

function openEditModal(c) {
    editingName = c.name;
    addModalTitle.textContent = 'Edit connection';
    addError.classList.add('hidden');
    addForm.reset();
    addForm.name.value = c.name;
    addForm.host.value = c.host;
    addForm.user.value = c.user;
    addForm.port.value = c.port;
    addForm.identityFile.value = c.identityFile || c.identity_file || '';
    addModal.classList.remove('hidden');
    addForm.name.focus();
}

addCancel.addEventListener('click', () => {
    addModal.classList.add('hidden');
});

addForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    addError.classList.add('hidden');

    const input = {
        name: addForm.name.value.trim(),
        host: addForm.host.value.trim(),
        user: addForm.user.value.trim(),
        port: parseInt(addForm.port.value, 10) || 22,
        identityFile: addForm.identityFile.value.trim(),
    };

    try {
        if (editingName) {
            await UpdateConnection(editingName, input);
        } else {
            await AddConnection(input);
        }
        addModal.classList.add('hidden');
        await refreshConnections();
    } catch (err) {
        addError.textContent = String(err);
        addError.classList.remove('hidden');
    }
});

updateEmptyState();
refreshConnections();
