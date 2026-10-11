# frozen_string_literal: true

require "ledger/account"
require "ledger/account/freezing"
require "ledger/reporting"
require "ledger/core_ext/string"

module Ledger
  VERSION = "1.4.0"

  class Error < StandardError; end
  class FrozenAccountError < Error; end
end
