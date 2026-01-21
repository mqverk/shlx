<script>
	let sessionId = $state('');
	let role = $state('readonly');
	let ownerToken = $state('');
	let creating = $state(false);
	let sessionInfo = $state(null);

	async function createSession() {
		creating = true;
		try {
			const res = await fetch('/api/create', { method: 'POST' });
			const data = await res.json();
			sessionInfo = data;
		} catch (e) {
			alert('Failed to create session: ' + e.message);
		} finally {
			creating = false;
		}
	}

	function joinSession() {
		if (!sessionId) {
			alert('Please enter a session ID');
			return;
		}
		const url = `/session/${sessionId}${ownerToken ? `?token=${ownerToken}` : `?role=${role}`}`;
		window.location.href = url;
	}

	function copyToClipboard(text) {
		navigator.clipboard.writeText(text);
	}
</script>

<div class="home-container">
	<div class="logo" style="font-size: 48px; margin-bottom: 16px;">shlx</div>
	<p style="color: var(--text-secondary); margin-bottom: 32px;">
		Fast, collaborative live terminal sharing
	</p>

	{#if sessionInfo}
		<div class="card">
			<h2>✓ Session Created</h2>
			<p>Share this session with others:</p>
			
			<div class="form-group">
				<label>Session ID</label>
				<div style="display: flex; gap: 8px;">
					<input type="text" value={sessionInfo.sessionId} readonly />
					<button class="copy-btn" onclick={() => copyToClipboard(sessionInfo.sessionId)}>
						Copy
					</button>
				</div>
			</div>

			<div class="form-group">
				<label>Session URL</label>
				<div style="display: flex; gap: 8px;">
					<input type="text" value={sessionInfo.url} readonly />
					<button class="copy-btn" onclick={() => copyToClipboard(sessionInfo.url)}>
						Copy
					</button>
				</div>
			</div>

			<div class="form-group">
				<label>Owner Token (keep private!)</label>
				<div style="display: flex; gap: 8px;">
					<input type="password" value={sessionInfo.ownerToken} readonly />
					<button class="copy-btn" onclick={() => copyToClipboard(sessionInfo.ownerToken)}>
						Copy
					</button>
				</div>
			</div>

			<button onclick={() => window.location.href = `/session/${sessionInfo.sessionId}?token=${sessionInfo.ownerToken}`}>
				Join Session
			</button>
		</div>
	{:else}
		<div class="card">
			<h2>Create Session</h2>
			<p>Start a new terminal session that others can join and collaborate in real-time.</p>
			<button onclick={createSession} disabled={creating}>
				{creating ? 'Creating...' : 'Create New Session'}
			</button>
		</div>

		<div class="card">
			<h2>Join Session</h2>
			<p>Enter a session ID to join an existing terminal session.</p>
			
			<div class="form-group">
				<label>Session ID</label>
				<input type="text" bind:value={sessionId} placeholder="Enter session ID" />
			</div>

			<div class="form-group">
				<label>Owner Token (optional)</label>
				<input type="password" bind:value={ownerToken} placeholder="Leave empty for read-only" />
			</div>

			{#if !ownerToken}
				<div class="form-group">
					<label>Role</label>
					<select bind:value={role} style="background: var(--bg-secondary); color: var(--text-primary); border: 1px solid var(--border); padding: 8px; border-radius: 6px; width: 100%;">
						<option value="readonly">Read-Only</option>
						<option value="interactive">Interactive</option>
					</select>
				</div>
			{/if}

			<button onclick={joinSession}>Join Session</button>
		</div>
	{/if}
</div>
