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

source vim/scip-tags.vim

let s:root = tempname() . ' project'
call mkdir(s:root . '/.git', 'p')
call mkdir(s:root . '/src')
let s:log = s:root . '/runs'
let g:scip_tags_update_command = 'mkdir running || { touch overlap; exit 1; }; '
      \ . 'pwd >> runs; sleep 0.3; rmdir running; touch tags'

function! s:AwaitRuns(count) abort
  for attempt in range(150)
    if filereadable(s:log) && len(readfile(s:log)) >= a:count &&
          \ !isdirectory(s:root . '/running')
      return
    endif
    sleep 20m
  endfor
endfunction

try
  execute 'edit ' . fnameescape(s:root . '/src/main.go')
  call setline(1, 'package main')
  write
  sleep 1650m
  call assert_false(filereadable(s:log), 'automatic refresh starts disabled')

  ScipTagsUpdate
  call s:AwaitRuns(1)
  call assert_equal([resolve(s:root)], readfile(s:log), 'manual refresh runs from the project root')

  let g:scip_tags_auto_update = 1
  write
  sleep 100m
  write
  call s:AwaitRuns(2)
  sleep 350m
  call assert_equal(2, len(readfile(s:log)), 'consecutive saves start one refresh')

  ScipTagsUpdate
  ScipTagsUpdate
  call s:AwaitRuns(4)
  call assert_equal(4, len(readfile(s:log)), 'requests during a job start one follow-up')
  call assert_false(filereadable(s:root . '/overlap'), 'updates for a project are serialized')

  call writefile(['previous tags'], s:root . '/tags')
  let b:scip_tags_update_command = 'echo failing-updater >&2; exit 17'
  ScipTagsUpdate
  for attempt in range(150)
    if execute('messages') =~# 'failing-updater'
      break
    endif
    sleep 20m
  endfor
  call assert_match('failing-updater', execute('messages'))
  call assert_equal(['previous tags'], readfile(s:root . '/tags'))
finally
  call delete(s:root, 'rf')
endtry

if !empty(v:errors)
  for failure in v:errors
    echomsg failure
  endfor
  cquit
endif
qa!
