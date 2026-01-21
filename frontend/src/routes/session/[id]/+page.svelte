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
	let showUsers = $state(true);
	let showNetwork = $state(true);
	let terminalSize = $state({ width: 900, height: 600 });
	let terminalPosition = $state({ x: 0, y: 0 });
	let isResizing = $state(false);
	let resizeEdge = $state(null);
	let isDragging = $state(false);
	let dragStart = $state({ x: 0, y: 0 });
	let showNamePrompt = $state(true);
	let userName = $state('');
	let userNameInput = $state('');

	onMount(() => {
		// Center terminal initially
		const centerX = (window.innerWidth - terminalSize.width) / 2;
		const centerY = (window.innerHeight - terminalSize.height) / 2 - 50; // -50 for header
		terminalPosition = { x: Math.max(0, centerX), y: Math.max(0, centerY) };

		initTerminal();
		
		window.addEventListener('resize', handleWindowResize);
		return () => {
			window.removeEventListener('resize', handleWindowResize);
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
				role: token ? 'owner' : role,
				name: userName || 'Guest'
			}));

			// Start ping interval
			setInterval(() => {
				if (ws && ws.readyState === WebSocket.OPEN) {
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
					role: msg.data.role,
					name: msg.data.name || 'Guest'
				};
				users = msg.data.users || [];
				sendResize();
				break;
			case 'user_joined':
				users = [...users, { 
					id: msg.data.userId, 
					role: msg.data.role,
					name: msg.data.name || 'Guest'
				}];
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

	function handleWindowResize() {
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

	function toggleUsers() {
		showUsers = !showUsers;
	}

	function toggleNetwork() {
		showNetwork = !showNetwork;
	}

	function startResize(edge, event) {
		isResizing = true;
		resizeEdge = edge;
		event.preventDefault();
		document.addEventListener('mousemove', handleResize);
		document.addEventListener('mouseup', stopResize);
	}

	function handleResize(event) {
		if (!isResizing || !resizeEdge) return;

		const container = document.querySelector('.terminal-window');
		if (!container) return;

		const rect = container.getBoundingClientRect();

		if (resizeEdge.includes('e')) {
			const newWidth = event.clientX - rect.left;
			terminalSize.width = Math.max(400, Math.min(1600, newWidth));
		}
		if (resizeEdge.includes('w')) {
			const newWidth = rect.right - event.clientX;
			terminalSize.width = Math.max(400, Math.min(1600, newWidth));
		}
		if (resizeEdge.includes('s')) {
			const newHeight = event.clientY - rect.top;
			terminalSize.height = Math.max(300, Math.min(1000, newHeight));
		}
		if (resizeEdge.includes('n')) {
			const newHeight = rect.bottom - event.clientY;
			terminalSize.height = Math.max(300, Math.min(1000, newHeight));
		}

		if (fitAddon && terminal) {
			setTimeout(() => {
				fitAddon.fit();
				sendResize();
			}, 0);
		}
	}

	function stopResize() {
		isResizing = false;
		resizeEdge = null;
		document.removeEventListener('mousemove', handleResize);
		document.removeEventListener('mouseup', stopResize);
	}

	function startDrag(event) {
		if (event.target.closest('.terminal-dots')) return;
		isDragging = true;
		dragStart = {
			x: event.clientX - terminalPosition.x,
			y: event.clientY - terminalPosition.y
		};
		event.preventDefault();
		document.addEventListener('mousemove', handleDrag);
		document.addEventListener('mouseup', stopDrag);
	}

	function handleDrag(event) {
		if (!isDragging) return;
		terminalPosition = {
			x: event.clientX - dragStart.x,
			y: event.clientY - dragStart.y
		};
	}

	function stopDrag() {
		isDragging = false;
		document.removeEventListener('mousemove', handleDrag);
		document.removeEventListener('mouseup', stopDrag);
	}

	function submitName() {
		if (userNameInput.trim()) {
			userName = userNameInput.trim();
			showNamePrompt = false;
			connectWebSocket();
		}
	}

	function skipName() {
		userName = 'Guest';
		showNamePrompt = false;
		connectWebSocket();
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
			<button class="icon-btn" onclick={toggleUsers} title="Toggle Users" style="background: {showUsers ? 'var(--bg-tertiary)' : 'transparent'}">
				<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
					<circle cx="9" cy="7" r="4"></circle>
					<path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
					<path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
				</svg>
			</button>
			<button class="icon-btn" onclick={toggleNetwork} title="Toggle Network Stats" style="background: {showNetwork ? 'var(--bg-tertiary)' : 'transparent'}">
				<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline>
				</svg>
			</button>
			<button class="copy-btn" onclick={copySessionUrl}>Share</button>
		</div>
	</div>

	<!-- Users Panel -->
	{#if showUsers}
		<div class="users-panel">
			<div class="users-title">Connected ({users.length})</div>
			<div class="users">
				{#each users as user}
					<div class="user-badge">
						<div class="user-dot"></div>
						<span class={getRoleColor(user.role)}>
							{user.id === currentUser?.id ? (currentUser?.name || 'You') : (user.name || 'Guest')}
						</span>
						<span style="color: var(--text-secondary); font-size: 11px;">({user.role})</span>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Sidebar with Stats -->
	{#if showNetwork}
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
	{/if}

	<div class="session-layout">
		<div class="terminal-window" style="width: {terminalSize.width}px; height: {terminalSize.height}px; position: absolute; left: {terminalPosition.x}px; top: {terminalPosition.y}px; cursor: {isDragging ? 'grabbing' : 'default'};">
			<!-- Resize handles -->
			<div class="resize-handle resize-n" onmousedown={(e) => startResize('n', e)}></div>
			<div class="resize-handle resize-s" onmousedown={(e) => startResize('s', e)}></div>
			<div class="resize-handle resize-e" onmousedown={(e) => startResize('e', e)}></div>
			<div class="resize-handle resize-w" onmousedown={(e) => startResize('w', e)}></div>
			<div class="resize-handle resize-ne" onmousedown={(e) => startResize('ne', e)}></div>
			<div class="resize-handle resize-nw" onmousedown={(e) => startResize('nw', e)}></div>
			<div class="resize-handle resize-se" onmousedown={(e) => startResize('se', e)}></div>
			<div class="resize-handle resize-sw" onmousedown={(e) => startResize('sw', e)}></div>

			<div class="terminal-titlebar" onmousedown={startDrag} style="cursor: {isDragging ? 'grabbing' : 'grab'};">
				<div class="terminal-dots">
					<div class="terminal-dot dot-red"></div>
					<div class="terminal-dot dot-yellow"></div>
					<div class="terminal-dot dot-green"></div>
				</div>
				<div class="terminal-title">{currentUser?.name || userName || 'Guest'}@{sessionId.slice(0, 8)}</div>
			</div>
			<div class="terminal-container" bind:this={terminalElement}></div>
		</div>
	</div>

	<!-- Name Prompt Modal -->
	{#if showNamePrompt}
		<div class="modal-overlay">
			<div class="modal">
				<h2>Welcome to shlx!</h2>
				<p>Choose a display name for this session:</p>
				<input 
					type="text" 
					bind:value={userNameInput} 
					placeholder="Enter your name" 
					onkeydown={(e) => e.key === 'Enter' && submitName()}
					autofocus
					style="margin: 16px 0;"
				/>
				<div class="button-group">
					<button onclick={submitName} disabled={!userNameInput.trim()}>Join</button>
					<button onclick={skipName} style="background: var(--bg-tertiary);">Skip</button>
				</div>
			</div>
		</div>
	{/if}
</div>
