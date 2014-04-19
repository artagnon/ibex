require 'net/http'
require 'json'
require 'ncursesw'

module Ncurses
  def rows
    lame, lamer = [], []
    stdscr.getmaxyx lame, lamer
    lame.first
  end

  def cols
    lame, lamer = [], []
    stdscr.getmaxyx lame, lamer
    lamer.first
  end

  def curx
    lame, lamer = [], []
    stdscr.getyx lame, lamer
    lamer.first
  end
  module_function :rows, :cols, :curx
end

class BufferManager
  def initialize
    @minibuf_stack = ["foom", "baz"]
    @textfields = {}
    @in_x = ENV["TERM"] =~ /(xterm|rxvt|screen)/
  end

  def draw_screen
    status, title = "status-bar", "title"

    ## http://rtfm.etla.org/xterm/ctlseq.html (see Operating System Controls)
    print "\033]0;#{title}\07" if title && @in_x

    draw_minibuf
    draw_inbox
    

    Ncurses.doupdate
    Ncurses.refresh
  end

  def draw_inbox
    dump = Net::HTTP.get('localhost', '/Inbox.json', 8080)
    h = JSON.parse(dump)
    h.keys.sort.each_with_index do |key, index|
      Ncurses.mvaddstr index, 0, h[key][0]["Subject"]
    end
    Ncurses.refresh
  end

  def draw_minibuf
    m = @minibuf_stack.compact
    m.each_with_index do |s, i|
      Ncurses.mvaddstr Ncurses.rows - i - 1, 0, s + (" " * [Ncurses.cols - s.length, 0].max)
    end
    Ncurses.refresh
  end
end

def start_cursing
  Ncurses.initscr
  Ncurses.noecho
  Ncurses.cbreak
  Ncurses.stdscr.keypad 1
  Ncurses.use_default_colors
  Ncurses.curs_set 0
  Ncurses.start_color
end

def stop_cursing
  Ncurses.curs_set 1
  Ncurses.echo
  Ncurses.endwin
end

start_cursing
bm = BufferManager.new
bm.draw_screen
while true
  char = Ncurses.getch
  if char == 113 # for 'q'
    stop_cursing
    break
  end
end
