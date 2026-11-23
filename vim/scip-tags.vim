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

function! s:Tag(pattern, flags, info) abort
  if a:flags !~# 'c' && get(a:info, 'user_data', '') !=# 'scip-tags'
    return v:null
  endif

  let source = resolve(expand('%:p'))
  let tags = findfile('tags', escape(fnamemodify(source, ':h'), ' ,;') . ';')
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
