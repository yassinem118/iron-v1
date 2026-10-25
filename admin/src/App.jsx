import { useCallback, useEffect, useState } from 'react';
import { Shield, RefreshCw, LogIn, Power, UserPlus, Pencil, X } from 'lucide-react';
import { listUsers, setActive, createUser, updateUser } from './api';

const blank = { accountId: '', password: '' };

export default function App() {
  const [key, setKey] = useState('');
  const [users, setUsers] = useState([]);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState(blank);
  const [editing, setEditing] = useState(null);

  const load = useCallback(async () => {
    if (!key) return;
    setBusy(true); setError('');
    try { const data = await listUsers(key); setUsers(data.users); }
    catch (e) { setError(e.message || 'Could not load accounts. Check the admin key and API server.'); }
    finally { setBusy(false); }
  }, [key]);

  useEffect(() => { if (key) load(); }, [key, load]);

  function startCreate() { setEditing(null); setForm(blank); setError(''); setNotice(''); }
  function startEdit(u) { setEditing(u); setForm({ accountId: u.accountId, password: '' }); setError(''); setNotice(''); }
  function cancelForm() { setEditing(null); setForm(blank); }

  async function save(e) {
    e.preventDefault(); setError(''); setNotice('');
    if (!key) return setError('Connect with the admin API key first.');
    if (!editing && form.password.length < 12) return setError('Password must be at least 12 characters.');
    if (form.password && form.password.length < 12) return setError('A new password must be at least 12 characters.');
    setBusy(true);
    try {
      if (editing) {
        const payload = { accountId: form.accountId.trim() };
        if (form.password) payload.password = form.password;
        await updateUser(key, editing._id, payload);
        setNotice(form.password ? 'Account updated and password reset.' : 'Account ID updated.');
      } else {
        await createUser(key, { accountId: form.accountId.trim(), password: form.password });
        setNotice('Account created.');
      }
      cancelForm();
      await load();
    } catch (e) { setError(e.message || 'Could not save account.'); }
    finally { setBusy(false); }
  }

  async function toggle(u) {
    try { await setActive(key, u._id, !u.active); await load(); }
    catch (e) { setError(e.message || 'Status update failed.'); }
  }

  return <main className="shell">
    <header className="top"><div className="brand"><div className="mark"><Shield size={23}/></div><div><b>IRON v1</b><small>CONTROL CENTER</small></div></div><span className="live"><i/> API ADMIN</span></header>
    <section className="intro"><p className="eyebrow">SYSTEM OVERVIEW / 01</p><h1>Master Dashboard</h1><p className="sub">Manage Iron v1 accounts and service status.</p></section>
    <section className="panel access"><label htmlFor="adminKey">Admin API key</label><div className="accessRow"><input id="adminKey" type="password" value={key} onChange={e=>setKey(e.target.value)} placeholder="Enter your admin key" autoComplete="current-password"/><button onClick={load} disabled={busy || !key}><LogIn size={16}/> Connect</button></div><p className="hint">Use a secret supplied through your deployment environment. Never commit it to source control.</p></section>
    <section className="stats"><div className="panel stat"><span>REGISTERED ACCOUNTS</span><strong>{users.length}</strong></div><div className="panel stat"><span>ACTIVE IN LIST</span><strong>{users.filter(u=>u.active).length}</strong></div><button className="refresh" onClick={load} disabled={busy || !key}><RefreshCw size={16} className={busy?'spin':''}/> Refresh</button></section>
    <section className="panel editor">
      <div className="editorHead"><div><h2>{editing ? 'Edit account' : 'Create account'}</h2><p>Set an account ID and password. Existing passwords cannot be viewed; leave the password blank when editing to keep it unchanged.</p></div>{editing && <button className="action" type="button" onClick={cancelForm}><X size={14}/> Cancel</button>}</div>
      <form onSubmit={save} className="accountForm">
        <label>USER / LICENSE ID<input required minLength={3} maxLength={64} pattern="[a-zA-Z0-9_.-]+" value={form.accountId} onChange={e=>setForm({...form,accountId:e.target.value})} placeholder="client_001" autoComplete="off"/></label>
        <label>{editing ? 'NEW PASSWORD (OPTIONAL)' : 'PASSWORD'}<input type="password" minLength={12} maxLength={128} required={!editing} value={form.password} onChange={e=>setForm({...form,password:e.target.value})} placeholder={editing?'Leave blank to keep current password':'At least 12 characters'} autoComplete="new-password"/></label>
        <button type="submit" disabled={busy || !key}><UserPlus size={16}/>{editing?'Save changes':'Create account'}</button>
      </form>
    </section>
    <section className="panel tablePanel"><div className="tableHead"><div><h2>Account registry</h2><p>Passwords are hashed and never displayed. Use Edit to reset a password.</p></div><span className="count">{users.length} RECORDS</span></div>
      {error && <div className="error" role="alert">{error}</div>}
      {notice && <div className="notice" role="status">{notice}</div>}
      <div className="tableWrap"><table><thead><tr><th>#</th><th>IRON ACCOUNT ID</th><th>CREATED</th><th>STATUS</th><th>ACTIONS</th></tr></thead><tbody>{users.map((u,i)=><tr key={u._id}><td>{String(i+1).padStart(2,'0')}</td><td className="account">{u.accountId}</td><td>{new Date(u.createdAt).toLocaleString()}</td><td><span className={`status ${u.active?'on':'off'}`}>{u.active?'ACTIVE':'DISABLED'}</span></td><td className="actions"><button className="action" onClick={()=>startEdit(u)}><Pencil size={14}/> Edit</button><button className="action" onClick={()=>toggle(u)}><Power size={14}/>{u.active?'Disable':'Enable'}</button></td></tr>)}{!users.length && <tr><td colSpan="5" className="empty">{key?'No accounts found.':'Connect with your admin key to load accounts.'}</td></tr>}</tbody></table></div>
    </section><footer>IRON v1 <span>SECURE ACCOUNT MANAGEMENT</span></footer>
  </main>;
}
