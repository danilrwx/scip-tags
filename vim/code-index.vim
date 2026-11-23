" Copyright 2026 Daniil Antoshin
"
" Licensed under the Apache License, Version 2.0 (the "License");
" you may not use this file except in compliance with the License.
" You may obtain a copy of the License at
"
"     http:"www.apache.org/licenses/LICENSE-2.0
"
" Unless required by applicable law or agreed to in writing, software
" distributed under the License is distributed on an "AS IS" BASIS,
" WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
" See the License for the specific language governing permissions and
" limitations under the License.

if exists('g:loaded_code_index')
  finish
endif
let g:loaded_code_index = 1

function! s:Tag(pattern, flags, info) abort
  return [s:destination]
endfunction

function! s:Definition() abort
  try
    let source = resolve(expand('%:p'))
    let tags = findfile('tags', escape(fnamemodify(source, ':h'), ' ,;') . ';')
    if empty(tags)
      throw 'SCIP tags file not found'
    endif
    let tags = fnamemodify(tags, ':p')
    let root = fnamemodify(tags, ':h') . '/'
    let path = strpart(source, strlen(root))
    let prefix = printf('__scip__:%s:%d:', path, line('.'))
    let linehash = sha256(getline('.'))[:11]
    let previous_tags = &l:tags
    try
      let &l:tags = escape(tags, ' ,\')
      let entries = taglist('^' . escape(prefix, '\.^$~[]*'))
    finally
      let &l:tags = previous_tags
    endtry

    let seen = {}
    let targets = []
    for entry in entries
      if strpart(entry.name, 0, strlen(prefix)) !=# prefix ||
            \ get(entry, 'sourcehash', '') !=# linehash ||
            \ col('.') < str2nr(strpart(entry.name, strlen(prefix))) ||
            \ col('.') >= str2nr(get(entry, 'end', '0'))
        continue
      endif
      let filename = resolve(fnamemodify(entry.filename, ':p'))
      if strpart(filename, 0, strlen(root)) !=# root
        let filename = resolve(root . entry.filename)
      endif
      if strpart(filename, 0, strlen(root)) !=# root || !filereadable(filename) ||
            \ entry.cmd !~# '^[1-9][0-9]*$'
        continue
      endif
      let target = {'filename': filename, 'line': str2nr(entry.cmd),
            \ 'column': str2nr(get(entry, 'column', '1'))}
      if target.column < 1
        continue
      endif
      let key = json_encode(target)
      if !has_key(seen, key)
        let seen[key] = 1
        call add(targets, target)
      endif
    endfor
    if empty(targets)
      throw 'No SCIP tag for the current cursor and source line'
    endif
    let choice = len(targets) == 1 ? 1 : inputlist(['Go to definition:'] +
          \ map(copy(targets), {i, t -> printf('%d. %s:%d:%d', i+1, strtrans(t.filename), t.line, t.column)}))
    if choice < 1 || choice > len(targets)
      return
    endif
    let target = targets[choice - 1]
    let filename = target.filename
    let origin = bufnr()
    let previous = &l:tagfunc
    let name = expand('<cword>')
    let s:destination = {'name': name, 'filename': filename, 'cmd': string(target.line)}
    let &l:tagfunc = expand('<SID>') . 'Tag'
    try
      execute 'tag ' . (empty(name) ? 'definition' : fnameescape(name))
      call cursor(target.line, target.column)
    finally
      call setbufvar(origin, '&tagfunc', previous)
    endtry
  catch
    echohl WarningMsg
    echomsg v:exception
    echohl None
  endtry
endfunction

command! CodeIndexDefinition call s:Definition()
nnoremap <silent> <Plug>(CodeIndexDefinition) :CodeIndexDefinition<CR>
