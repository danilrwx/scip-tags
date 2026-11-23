" Copyright 2026 Daniil Antoshin
"
" Licensed under the Apache License, Version 2.0 (the "License");
" you may not use this file except in compliance with the License.
" You may obtain a copy of the License at
"
"     http://www.apache.org/licenses/LICENSE-2.0
"
" Unless required by applicable law or agreed to in writing, software
" distributed under the License is distributed on an "AS IS" BASIS,
" WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
" See the License for the specific language governing permissions and
" limitations under the License.

if exists('g:loaded_scip_tags')
  finish
endif
let g:loaded_scip_tags = 1
let s:updates = {}

function! s:SearchPath(path) abort
  let directory = fnamemodify(resolve(a:path), ':h')
  return escape(directory, ' ,;') . ';'
endfunction

function! s:Tag(pattern, flags, info) abort
  if a:flags !~# 'c' && get(a:info, 'user_data', '') !=# 'scip-tags'
    return v:null
  endif

  let source = resolve(expand('%:p'))
  let tags = findfile('tags', s:SearchPath(source))
  if empty(tags)
    return v:null
  endif

  let tags = fnamemodify(tags, ':p')
  let root = fnamemodify(tags, ':h') . '/'
  let prefix = printf('__scip__:%s:%d:', strpart(source, strlen(root)), line('.'))
  let previous_tags = &l:tags
  try
    let &l:tags = escape(tags, ' ,\')
    let entries = taglist('^' . escape(prefix, '\.^$~[]*'))
  finally
    let &l:tags = previous_tags
  endtry

  let column = col('.')
  let linehash = sha256(getline('.'))[:11]
  let targets = []
  let indexed = 0
  for entry in entries
    if strpart(entry.name, 0, strlen(prefix)) !=# prefix ||
          \ column < str2nr(strpart(entry.name, strlen(prefix))) ||
          \ column >= str2nr(get(entry, 'end', '0'))
      continue
    endif
    let indexed = 1
    if get(entry, 'sourcehash', '') !=# linehash || entry.cmd !~# '^[1-9][0-9]*$'
      continue
    endif
    let filename = resolve(fnamemodify(entry.filename, ':p'))
    if strpart(filename, 0, strlen(root)) !=# root
      let filename = resolve(root . entry.filename)
    endif
    if strpart(filename, 0, strlen(root)) !=# root || !filereadable(filename)
      continue
    endif
    let target_column = str2nr(get(entry, 'column', '1'))
    if target_column > 0
      let target = {'name': a:pattern, 'filename': filename, 'user_data': 'scip-tags',
            \ 'cmd': printf('/\%%%dl\%%%dc/', str2nr(entry.cmd), target_column)}
      if index(targets, target) == -1
        call add(targets, target)
      endif
    endif
  endfor
  " An outdated position must not fall back to a same-name definition.
  return indexed ? targets : v:null
endfunction

function! s:UpdateFinished(root, output, job, status, ...) abort
  let update = s:updates[a:root]
  let update.running = 0
  unlet! update.job
  if a:status != 0
    echohl WarningMsg
    echomsg 'scip-tags: update failed in ' . a:root . ' (exit ' . a:status . ')'
    if filereadable(a:output)
      for line in readfile(a:output, '', -3)
        echomsg line
      endfor
    endif
    echohl None
  endif
  call delete(a:output)

  if update.timer != -1
    return
  endif
  if update.queued
    call s:StartUpdate(a:root, -1)
  else
    call remove(s:updates, a:root)
  endif
endfunction

function! s:StartUpdate(root, timer) abort
  let update = s:updates[a:root]
  let update.timer = -1
  if update.running
    let update.queued = 1
    return
  endif
  let update.running = 1
  let update.queued = 0
  let output = tempname()
  let args = ['sh', '-c', "exec </dev/null >\"$1\" 2>&1\nshift\n" . update.command, 'scip-tags', output]
  let Done = function('s:UpdateFinished', [a:root, output])
  try
    if has('nvim')
      let failed = jobstart(args, {'cwd': a:root, 'on_exit': Done}) <= 0
    else
      let job = job_start(args, {'cwd': a:root, 'exit_cb': Done})
      let update.job = job
      let failed = job_status(job) ==# 'fail'
    endif
  catch /^Vim/
    let failed = 1
  endtry
  if failed
    call s:UpdateFinished(a:root, output, 0, 127)
  endif
endfunction

function! s:Update(buffer, path, delay) abort
  let command = getbufvar(a:buffer, 'scip_tags_update_command', get(g:, 'scip_tags_update_command', ''))
  if empty(command) || (!has('nvim') && !has('job'))
    if a:delay == 0
      echohl WarningMsg
      echomsg empty(command) ? 'scip-tags: set g:scip_tags_update_command' : 'scip-tags: job support is required'
      echohl None
    endif
    return
  endif

  let search = s:SearchPath(a:path)
  let marker = findfile('tags', search)
  if empty(marker)
    let marker = finddir('.git', search)
  endif
  if empty(marker)
    let marker = findfile('.git', search)
  endif
  let root = resolve(fnamemodify(empty(marker) ? getcwd() : fnamemodify(marker, ':h'), ':p'))
  let update = get(s:updates, root, {'timer': -1, 'running': 0, 'queued': 0})
  let s:updates[root] = update
  let update.command = command
  let update.queued = 0
  if update.timer != -1
    call timer_stop(update.timer)
  endif
  if a:delay > 0
    let update.timer = timer_start(a:delay, function('s:StartUpdate', [root]))
  else
    call s:StartUpdate(root, -1)
  endif
endfunction

function! s:AutoUpdate(buffer, path) abort
  if exists('*timer_start') && getbufvar(a:buffer, '&buftype') ==# '' &&
        \ getbufvar(a:buffer, 'scip_tags_auto_update', get(g:, 'scip_tags_auto_update', 0))
    call s:Update(a:buffer, a:path, 1500)
  endif
endfunction

if empty(&g:tagfunc)
  let &g:tagfunc = expand('<SID>') . 'Tag'
  if empty(&l:tagfunc)
    let &l:tagfunc = &g:tagfunc
  endif
endif

command! ScipTagsDefinition execute "normal! \<C-]>"
nnoremap <Plug>(ScipTagsDefinition) <C-]>

command! CodeIndexDefinition ScipTagsDefinition
nmap <Plug>(CodeIndexDefinition) <Plug>(ScipTagsDefinition)

command! ScipTagsUpdate call s:Update(bufnr('%'), expand('%:p'), 0)

augroup scip_tags_update
  autocmd!
  autocmd BufWritePost * call s:AutoUpdate(str2nr(expand('<abuf>')), expand('<afile>:p'))
augroup END
