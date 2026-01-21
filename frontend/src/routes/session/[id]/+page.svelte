<script>
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { Terminal } from '@xterm/xterm';
	import { FitAddon } from '@xterm/addon-fit';
	import { WebLinksAddon } from '@xterm/addon-web-links';
	import '@xterm/xterm/css/xterm.css';

	let terminalElement = $state(null);
	let terminal = null;
	let fitAddon = null;
	let ws = null;
	let connectionStatus = $state('connecting');
	let users = $state([]);
	let currentUser = $state(null);
	let sessionId = $derived($page.params.id);
	let token = $derived($page.url.searchParams.get('token'));
	let role = $derived($page.url.searchParams.get('role') || 'readonly');
	let latency = $state({ total: 0, server: 0, shell: 0 });

	onMount(() => {
		initTerminal();
		connectWebSocket();

		window.addEventListener('resize', handleResize);
		return () => {
			window.removeEventListener('resize', handleResize);
		};
	});

	onDestroy(() => {
		if (ws) ws.close();
		if (terminal) terminal.dispose();
	});

	function initTerminal() {
		terminal = new Terminal({
			cursorBlink: true,
			fontSize: 14,
			fontFamily: '"Cascadia Code", Menlo, Monaco, "Courier New", monospace',
			theme: {
				background: '#0d1117',
				foreground: '#e6edf3',
				cursor: '#58a6ff',
				cursorAccent: '#0d1117',
				black: '#484f58',
				red: '#ff7b72',
				green: '#3fb950',
				yellow: '#d29922',
				blue: '#58a6ff',
				magenta: '#bc8cff',
				cyan: '#39c5cf',
				white: '#b1bac4',
				brightBlack: '#6e7681',
				brightRed: '#ffa198',
				brightGreen: '#56d364',
				brightYellow: '#e3b341',
				brightBlue: '#79c0ff',
				brightMagenta: '#d2a8ff',
				brightCyan: '#56d4dd',
				brightWhite: '#f0f6fc'
			}
		});

		fitAddon = new FitAddon();
		terminal.loadAddon(fitAddon);
		terminal.loadAddon(new WebLinksAddon());

		terminal.open(terminalElement);
		fitAddon.fit();

		terminal.onData((data) => {
			if (ws && ws.readyState === WebSocket.OPEN) {
				ws.send(data);
			}
		});
	}

	function connectWebSocket() {
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const wsUrl = `${protocol}//${window.location.host}/ws`;
		
		ws = new WebSocket(wsUrl);
		ws.binaryType = 'arraybuffer';

		ws.onopen = () => {
			connectionStatus = 'connected';
			ws.send(JSON.stringify({
				sessionId,
				token: token || '',
				role: token ? 'owner' : role
			}));

			// Ping for latency measurement
			setInterval(() => {
				if (ws.readyState === WebSocket.OPEN) {
					const pingTime = Date.now();
					ws.send(JSON.stringify({ type: 'ping', time: pingTime }));
				}
			}, 2000);
		};

		ws.onmessage = (event) => {
			if (event.data instanceof ArrayBuffer) {
				const text = new TextDecoder().decode(event.data);
				terminal.write(text);
			} else {
				try {
					const msg = JSON.parse(event.data);
					handleControlMessage(msg);
				} catch (e) {
					console.error('Failed to parse message:', e);
				}
			}
		};

		ws.onerror = (error) => {
			console.error('WebSocket error:', error);
			connectionStatus = 'error';
		};

		ws.onclose = () => {
			connectionStatus = 'disconnected';
			terminal.write('\r\n\x1b[31mConnection closed\x1b[0m\r\n');
		};
	}

	function handleControlMessage(msg) {
		switch (msg.type) {
			case 'welcome':
				currentUser = {
					id: msg.data.userId,
					role: msg.data.role
				};
				users = msg.data.users || [];
				sendResize();
				break;
			case 'user_joined':
				users = [...users, { id: msg.data.userId, role: msg.data.role }];
				break;
			case 'user_left':
				users = users.filter(u => u.id !== msg.data.userId);
				break;
			case 'pong':
				const now = Date.now();
				const rtt = now - msg.data.time;
				latency = {
					total: rtt,
					server: rtt / 2,
					shell: rtt / 2
				};
				break;
			case 'error':
				alert('Error: ' + msg.data.message);
				break;
		}
	}

	function handleResize() {
		if (fitAddon) {
			fitAddon.fit();
			sendResize();
		}
	}

	function sendResize() {
		if (ws && ws.readyState === WebSocket.OPEN && terminal) {
			ws.send(JSON.stringify({
				type: 'resize',
				data: {
					rows: terminal.rows,
					cols: terminal.cols
				}
			}));
		}
	}

	function copySessionUrl() {
		const url = window.location.origin + '/session/' + sessionId;
		navigator.clipboard.writeText(url);
	}

	function getRoleColor(role) {
		switch (role) {
			case 'owner': return 'role-owner';
			case 'interactive': return 'role-interactive';
			default: return 'role-readonly';
		}
	}

	function formatLatency(ms) {
		if (ms < 1) return '<1 ms';
		return `${Math.round(ms)} ms`;
	}
</script>

<div id="app">
<div class="header">
<div class="header-left">
<a href="/" class="logo" style="text-decoration: none;">shlx</a>
<div class="connection-status {connectionStatus}">
<div class="user-dot" style="background: {connectionStatus === 'connected' ? 'var(--success)' : 'var(--error)'}"></div>
{connectionStatus === 'connected' ? 'You are connected!' : connectionStatus}
</div>
</div>

<div style="display: flex; gap: 12px; align-items: center;">
<button class="copy-btn" onclick={copySessionUrl}>Share</button>
</div>
</div>

<!-- Users Panel -->
<div class="users-panel">
<div class="users-title">Connected ({users.length})</div>
<div class="users">
{#each users as user}
<div class="user-badge">
<div class="user-dot"></div>
<span class={getRoleColor(user.role)}>
{user.id === currentUser?.id ? 'You' : user.id.slice(0, 8)}
</span>
<span style="color: var(--text-secondary); font-size: 11px;">({user.role})</span>
</div>
{/each}
</div>
</div>

<!-- Sidebar with Stats -->
<div class="sidebar">
<div class="panel">
<div class="panel-title">Network</div>
<div class="latency-info">
<div class="latency-row">
<span class="latency-label">Total latency</span>
<span class="latency-value">{formatLatency(latency.total)}</span>
</div>
<div class="latency-row">
<span class="latency-label">Server ↔ You</span>
<span class="latency-value">{formatLatency(latency.server)}</span>
</div>
<div class="latency-row">
<span class="latency-label">Shell ↔ Server</span>
<span class="latency-value">{formatLatency(latency.shell)}</span>
</div>
</div>
</div>
</div>

<div class="session-layout">
<div class="terminal-window">
<div class="terminal-titlebar">
<div class="terminal-dots">
<div class="terminal-dot dot-red"></div>
<div class="terminal-dot dot-yellow"></div>
<div class="terminal-dot dot-green"></div>
</div>
<div class="terminal-title">{currentUser?.id?.slice(0, 8) || 'guest'}@{sessionId.slice(0, 8)}</div>
</div>
<div class="terminal-container" bind:this={terminalElement}></div>
</div>
</div>
</div>
