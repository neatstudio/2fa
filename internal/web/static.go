package web

const pageHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>2FA</title>
<style>
:root{color-scheme:dark light;font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}body{margin:0;background:#111827;color:#e5e7eb}main{max-width:1100px;margin:0 auto;padding:24px}h1{margin:0 0 16px;font-size:28px}.panel{background:#1f2937;border:1px solid #374151;border-radius:12px;padding:16px;margin-bottom:16px}label{display:block;font-size:13px;color:#9ca3af;margin-bottom:6px}input,select,button{font:inherit;border-radius:8px;border:1px solid #4b5563;background:#111827;color:#f9fafb;padding:9px 10px}button{cursor:pointer;background:#2563eb;border-color:#2563eb}button.secondary{background:#374151;border-color:#4b5563}button.danger{background:#b91c1c;border-color:#b91c1c}.grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px}.actions{display:flex;gap:8px;align-items:end}.status{color:#9ca3af;font-size:13px;margin:8px 0 0}table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:10px;border-bottom:1px solid #374151}th{color:#9ca3af;font-size:12px;text-transform:uppercase}.code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:24px;letter-spacing:.08em}.remaining{font-variant-numeric:tabular-nums}.hidden{display:none}@media(max-width:800px){.grid{grid-template-columns:1fr}table{font-size:14px}.code{font-size:18px}}
</style>
</head>
<body>
<main>
<h1>2FA</h1>
<section class="panel">
<form id="add-form">
<div class="grid">
<div><label for="name">Name</label><input id="name" name="name" required autocomplete="off" spellcheck="false"></div>
<div><label for="secret">Secret</label><input id="secret" name="secret" required type="password" autocomplete="off" spellcheck="false"></div>
<div><label for="group">Group</label><input id="group" name="group" placeholder="default" autocomplete="off" spellcheck="false"></div>
<div><label for="note">Note</label><input id="note" name="note" autocomplete="off" spellcheck="false"></div>
</div>
<div class="actions" style="margin-top:12px"><button type="submit">Add account</button><span id="message" class="status"></span></div>
</form>
</section>
<section class="panel">
<div class="actions" style="justify-content:space-between;margin-bottom:12px">
<div><label for="filter">Group filter</label><select id="filter"><option value="">All</option></select></div>
<div class="status" id="connection">connecting...</div>
</div>
<table>
<thead><tr><th>Group</th><th>Name</th><th>Code</th><th>Remaining</th><th>Note</th><th>Actions</th></tr></thead>
<tbody id="accounts"><tr><td colspan="6">No accounts</td></tr></tbody>
</table>
</section>
<section id="edit-panel" class="panel hidden">
<h2>Edit <span id="edit-title"></span></h2>
<form id="edit-form">
<input type="hidden" id="edit-name">
<div class="grid">
<div><label for="edit-group">Group</label><input id="edit-group" autocomplete="off" spellcheck="false"></div>
<div><label for="edit-note">Note</label><input id="edit-note" autocomplete="off" spellcheck="false"></div>
<div><label for="edit-secret">New secret (optional)</label><input id="edit-secret" type="password" autocomplete="off" spellcheck="false"></div>
<div class="actions"><button type="submit">Save</button><button type="button" class="secondary" id="cancel-edit">Cancel</button></div>
</div>
</form>
</section>
</main>
<script>
const state={accounts:[],groups:[],events:null};
const csrf={'X-2FA-CSRF':'1','Content-Type':'application/json'};
function text(el,value){el.textContent=value==null?'':String(value)}
function selectedGroup(){return document.getElementById('filter').value}
function showMessage(value){text(document.getElementById('message'),value);setTimeout(()=>text(document.getElementById('message'),''),3000)}
function render(data){state.accounts=data.accounts||[];state.groups=data.groups||[];const filter=document.getElementById('filter');const current=filter.value;filter.replaceChildren(new Option('All',''));for(const g of state.groups){filter.appendChild(new Option(g,g))}filter.value=current;const tbody=document.getElementById('accounts');tbody.replaceChildren();if(state.accounts.length===0){const tr=document.createElement('tr');const td=document.createElement('td');td.colSpan=6;text(td,'No accounts');tr.appendChild(td);tbody.appendChild(tr);return}for(const account of state.accounts){const tr=document.createElement('tr');for(const value of [account.group,account.name]){const td=document.createElement('td');text(td,value);tr.appendChild(td)}const code=document.createElement('td');code.className='code';text(code,account.code);tr.appendChild(code);const remaining=document.createElement('td');remaining.className='remaining';text(remaining,account.remaining+'s');tr.appendChild(remaining);const note=document.createElement('td');text(note,account.note);tr.appendChild(note);const actions=document.createElement('td');const edit=document.createElement('button');edit.className='secondary';text(edit,'Edit');edit.onclick=()=>openEdit(account);const del=document.createElement('button');del.className='danger';text(del,'Delete');del.onclick=()=>deleteAccount(account.name);actions.append(edit,' ',del);tr.appendChild(actions);tbody.appendChild(tr)}}
async function api(path,options={}){const res=await fetch(path,{credentials:'same-origin',...options});if(!res.ok){throw new Error(await res.text()||res.statusText)}return res.status===204?null:res.json()}
function connect(){if(state.events){state.events.close()}const group=encodeURIComponent(selectedGroup());state.events=new EventSource('/api/events'+(group?'?group='+group:''));state.events.onopen=()=>text(document.getElementById('connection'),'live');state.events.onerror=()=>text(document.getElementById('connection'),'reconnecting...');state.events.addEventListener('accounts',e=>render(JSON.parse(e.data)))}
document.getElementById('filter').onchange=connect;
document.getElementById('add-form').onsubmit=async e=>{e.preventDefault();const form=e.currentTarget;try{await api('/api/accounts',{method:'POST',headers:csrf,body:JSON.stringify({name:form.name.value,secret:form.secret.value,group:form.group.value,note:form.note.value})});form.reset();showMessage('added')}catch(err){showMessage(err.message)}};
function openEdit(account){document.getElementById('edit-panel').classList.remove('hidden');text(document.getElementById('edit-title'),account.name);document.getElementById('edit-name').value=account.name;document.getElementById('edit-group').value=account.group;document.getElementById('edit-note').value=account.note;document.getElementById('edit-secret').value=''}
document.getElementById('cancel-edit').onclick=()=>document.getElementById('edit-panel').classList.add('hidden');
document.getElementById('edit-form').onsubmit=async e=>{e.preventDefault();const name=document.getElementById('edit-name').value;const body={group:document.getElementById('edit-group').value,note:document.getElementById('edit-note').value};const secret=document.getElementById('edit-secret').value;if(secret){body.secret=secret}try{await api('/api/accounts/'+encodeURIComponent(name),{method:'PATCH',headers:csrf,body:JSON.stringify(body)});document.getElementById('edit-panel').classList.add('hidden');showMessage('saved')}catch(err){showMessage(err.message)}};
async function deleteAccount(name){if(!confirm('Delete '+name+'?'))return;try{await api('/api/accounts/'+encodeURIComponent(name),{method:'DELETE',headers:{'X-2FA-CSRF':'1'}});showMessage('deleted')}catch(err){showMessage(err.message)}}
connect();
</script>
</body>
</html>`
